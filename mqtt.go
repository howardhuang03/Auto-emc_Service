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
)

var (
  mqttChan chan string
)

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
    if (c.Sensors > 15) { mapping = (mapping >> 4) }
    for index = 0; index < 4; index++ { // Skip device name
      if (mapping & (1 << index)) == 0 { s[index + 1] = "0" }
    }

    // Build string for thingspeak
    buffer.Reset()
    for i, v := range s {
      if i != 0 {buffer.WriteString(",")}
      // Sensors > 15 means we want to use field5 ~ field8 on thingspeak
      if (c.Sensors > 15 && i == 1) { buffer.WriteString("0,0,0,0,") }
      buffer.WriteString(v)
    }
    mqttChan <- buffer.String()

    // Build string for file writing
    buffer.Reset()
    for i, v := range s {
      if i != 0 {buffer.WriteString(",")}
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
    if (i > 0 && v != "0") { // Skip zero value & device name
      if buf.Len() > 0 {buf.WriteString("&")}
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

func mqttService() {
  mqttChan = make(chan string)
  cloudCli := mqttClientMaker(cloudUrl, cloudId)
  localCli := mqttClientMaker(localUrl, localId)

  setSubscriber(localCli, localDataTopic, f)

  for {
    setPublisher(cloudCli, <- mqttChan)
  }
}
