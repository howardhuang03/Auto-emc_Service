package main

import (
  "os"
  "fmt"
  "bytes"
  "strings"

  MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
  cloudUrl = "tcp://mqtt.thingspeak.com:1883"
  cloudId = "go-cloud"
  localUrl = "tcp://127.0.0.1:1883"
  localId = "go-local"
  localDataTopic = "channels/local/data"
  localCmdTopic = "channels/local/cmd"
  cloudUpdateCount = 2  // cloudUpdateCount * 5min
  dataSaveCount = 2  // dataSaveCount * 5min
)

var (
  mqttChan chan string
  cloudCount int
  dataCount int
)

// Define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
  fmt.Printf("Received: TOPIC: %s, MSG: %s\n", msg.Topic(), msg.Payload())

  var buffer bytes.Buffer
  buffer.Write(msg.Payload())
  s := strings.Split(buffer.String(), ",")
  cloudCount++
  dataCount++

  // Check the device is existed in config
  if _, ok := configMaps[s[0]]; !ok {
    fmt.Printf("Device: %s is not found!!\n", s[0])
    return
  }

  // Update data to cloud once meet the limit
  if (cloudCount == cloudUpdateCount) {
    mqttChan <- buffer.String()
    cloudCount = 0
  }

  // Save data to file once meet the limit
  if (dataCount == dataSaveCount) {
    mainChan <- buffer.String()
    dataCount = 0
  }
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
  for i := range s {
    if (i == 0) {continue} // Skip device name
    if (s[i] != "0") { // Skip zero value
      tmp := fmt.Sprintf("field%d=%s", i, s[i])
      buf.WriteString(tmp)
      if i != cap(s) - 1 {buf.WriteString("&")}
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

func mqttService() {
  mqttChan = make(chan string)
  cloudCli := mqttClientMaker(cloudUrl, cloudId)
  localCli := mqttClientMaker(localUrl, localId)
  cloudCount = cloudUpdateCount - 1 // Update first data
  dataCount = dataSaveCount - 1 // Record first data

  setSubscriber(localCli, localDataTopic, f)

  for {
    setPublisher(cloudCli, <- mqttChan)
  }
}
