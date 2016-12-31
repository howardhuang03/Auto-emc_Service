package main

import (
  "os"
  "io"
  "fmt"
  "time"
  "flag"
  "bytes"
  "strings"
  "io/ioutil"
  "encoding/json"

  MQTT "github.com/eclipse/paho.mqtt.golang"
)

type config struct {
  Device string `json:"device"`
  Id string `json:"id"`
  Key string `json:"key"`
}

const (
  configFile = "./config.json"
  cloudUrl = "tcp://mqtt.thingspeak.com:1883"
  cloudId = "go-cloud"
  localUrl = "tcp://127.0.0.1:1883"
  localId = "go-local"
  localDataTopic = "channels/local/data"
  localCmdTopic = "channels/local/cmd"
)

var (
  messages chan string
  configMaps map[string]config
  // Flag for argument input
  configDir = flag.String("config", "./config.json", "dir to config file")
)

func check(e error) {
  if e != nil {
      panic(e)
  }
}

func filePrefix() string {
	ts := time.Now().Format("2006-01-02-15:04:05.00")
	ts = strings.Replace(ts, ".", ":", 1)
	return ts
}

// Define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
  fmt.Printf("Received: TOPIC: %s, MSG: %s\n", msg.Topic(), msg.Payload())

  // Write incoming data to file
  var buffer bytes.Buffer
  buffer.WriteString(filePrefix())
  buffer.WriteString(",")
  buffer.Write(msg.Payload())
  buffer.WriteString("\n")
  n3, err := localFile.Write(buffer.Bytes())
  _ = n3
  check(err)
  localFile.Sync()

  // Update new data
  buffer.Reset()
  buffer.Write(msg.Payload())
  messages <- buffer.String()
}

func mqttClientMaker(url string, id string) MQTT.Client {
  opts := MQTT.NewClientOptions().AddBroker(url)
  opts.SetClientID(id)
  opts.SetDefaultPublishHandler(f)

  c := MQTT.NewClient(opts)
  if token := c.Connect(); token.Wait() && token.Error() != nil {
    panic(token.Error())
  }

  return c
}

func setSubscriber(c MQTT.Client, topic string, f MQTT.MessageHandler) {
  if token := c.Subscribe(topic, 0, f); token.Wait() && token.Error() != nil {
    fmt.Println(token.Error())
    os.Exit(1)
  }
}

func setPublisher(c MQTT.Client, msg string) {
  var buf bytes.Buffer

  // Build thingspeak data string
  s := strings.Split(msg, ",")
  topic := setTopic(configMaps[s[0]].Id, configMaps[s[0]].Key)
  for i := range s {
    if (i == 0) {continue} // Skip device name
    tmp := fmt.Sprintf("field%d=%s", i, s[i])
    buf.WriteString(tmp)
    if i != cap(s) - 1 {
      buf.WriteString("&")
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
    maps[c.Device] = c
  }

  fmt.Println("maps:", maps)
  return maps
}

func mqttService() {
  configMaps = setConfigMaps(configFile)
  messages = make(chan string)
  cloudCli := mqttClientMaker(cloudUrl, cloudId)
  localCli := mqttClientMaker(localUrl, localId)

  setSubscriber(localCli, localDataTopic, f)

  for {
    setPublisher(cloudCli, <- messages)
  }
}
