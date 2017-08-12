package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	cloudUrl       = "tcp://mqtt.thingspeak.com:1883"
	cloudId        = "go-cloud"
	localUrl       = "tcp://127.0.0.1:1883"
	localId        = "go-local"
	localDataTopic = "channels/local/data"
	localCmdTopic  = "channels/local/cmd"
)

var (
	configMaps  map[string]config
	devConfMaps map[string]devConf
	mqttChan    chan string
)

type config struct {
	Device    string `json:"device"`
	Id        string `json:"id"`
	Key       string `json:"key"`
	Interval  int    `json:interval` // Update per interval * 5min
	Sensors   int    `json:sensors`
	localFile string
}

type devConf struct {
	Count int // Interval check
}

// Define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Received: TOPIC: %s, MSG: %s\n", msg.Topic(), msg.Payload())

	var buffer bytes.Buffer
	buffer.Write(msg.Payload())
	s := strings.Split(buffer.String(), ",")

	// Check the device is existed in config
	c, ok := configMaps[s[0]]
	if !ok {
		fmt.Printf("Device: %s is not found!!\n", s[0])
		return
	}

	// Increse the count for update and write file
	m := devConfMaps[s[0]]
	m.Count++

	// Update data to cloud and save data to file once meet the limit
	if m.Count == c.Interval {
		// Check enabled sensor mapping
		var index uint
		var mapping = c.Sensors
		if c.Sensors > 15 {
			mapping = (mapping >> 4)
		}
		for index = 0; index < 4; index++ { // Skip device name
			if (mapping & (1 << index)) == 0 {
				s[index+1] = "0"
			}
		}

		// Build string for thingspeak
		buffer.Reset()
		for i, v := range s {
			if i != 0 {
				buffer.WriteString(",")
			}
			// Sensors > 15 means we want to use field5 ~ field8 on thingspeak
			if c.Sensors > 15 && i == 1 {
				buffer.WriteString("0,0,0,0,")
			}
			buffer.WriteString(v)
		}
		mqttChan <- buffer.String()

		// Build string for file writing
		buffer.Reset()
		for i, v := range s {
			if i != 0 {
				buffer.WriteString(",")
			}
			buffer.WriteString(v)
		}
		mainChan <- buffer.String()
		m.Count = 0
	}

	devConfMaps[s[0]] = m
}

func mqttClientMaker(url string, id string) MQTT.Client {
	opts := MQTT.NewClientOptions().AddBroker(url)
	opts.SetClientID(id)
	opts.SetDefaultPublishHandler(f)

	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("mqtt client build fail, url: %s, id: %s\n", url, id)
		fmt.Println(token.Error())
		// Return an empty client
		c = nil
	} else {
		fmt.Printf("mqtt client build success, url: %s, id: %s\n", url, id)
	}

	return c
}

func setSubscriber(c MQTT.Client, topic string, f MQTT.MessageHandler) {
	if c == nil {
		fmt.Printf("Can't use empty client to create subscriber: %s\n", topic)
		return
	}

	if token := c.Subscribe(topic, 0, f); token.Wait() && token.Error() != nil {
		fmt.Printf("Create subscriber error, topic: %s\n", topic)
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func setPublisher(c MQTT.Client, msg string) {
	var buf bytes.Buffer

	if c == nil {
		fmt.Printf("Can't use empty client to send msg:%s to cloud\n", msg)
		return
	}

	// Build thingspeak data string
	s := strings.Split(msg, ",")
	topic := setTopic(configMaps[s[0]].Id, configMaps[s[0]].Key)
	for i, v := range s {
		if i > 0 && v != "0" { // Skip zero value & device name
			if buf.Len() > 0 {
				buf.WriteString("&")
			}
			tmp := fmt.Sprintf("field%d=%s", i, v)
			buf.WriteString(tmp)
		}
	}

	// Publish message
	token := c.Publish(topic, 0, false, buf.String())
	fmt.Printf("Pulished: TOPIC: %s, MSG: %s\n", topic, buf.String())
	token.Wait()
}

func setTopic(id string, key string) string {
	s := fmt.Sprintf("channels/%s/publish/%s", id, key)
	return s
}

func setConfigMaps(file string) map[string]config {
	var c config
	maps := make(map[string]config)

	jsonData, e := ioutil.ReadFile(file)
	if e != nil {
		check(e)
		os.Exit(1)
	}

	jsonParser := json.NewDecoder(bytes.NewReader(jsonData))
	for {
		if err := jsonParser.Decode(&c); err == io.EOF {
			break
		} else if err != nil {
			check(e)
		}

		if err := os.MkdirAll(c.Device, 0777); err != nil {
			fmt.Println("Mkdir %s failed: %v", c.Device, err)
		}

		fname := fmt.Sprintf("%s/%s.csv", c.Device, time.Now().Format("20060102"))
		file, err := os.Create(fname)
		if err != nil {
			fmt.Println("create %s fail, err: %v", fname, err)
		}

		c.localFile = fname
		defer file.Close()

		maps[c.Device] = c
	}

	fmt.Println("maps:", maps)
	return maps
}

func setDevConfMaps(m map[string]config) map[string]devConf {
	var c devConf
	maps := make(map[string]devConf)

	for k, v := range m {
		c.Count = v.Interval - 1 // Update first data
		maps[k] = c
	}

	return maps
}

func mqttService() {
	// Initialize related config maps
	configMaps = setConfigMaps(*configDir)
	devConfMaps = setDevConfMaps(configMaps)
	// Initialize mqtt client
	mqttChan = make(chan string)
	cloudCli := mqttClientMaker(cloudUrl, cloudId)
	localCli := mqttClientMaker(localUrl, localId)

	setSubscriber(localCli, localDataTopic, f)

	for {
		setPublisher(cloudCli, <-mqttChan)
	}
}
