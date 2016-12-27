package main

import (
  "os"
  "fmt"
  "time"
  "bytes"
  "strings"

  MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
  cloudUrl = "tcp://mqtt.thingspeak.com:1883"
  cloudId = "go-cloud"
  channelId = "XXXXX"
  channelKey = "XXXXX"
  localUrl = "tcp://127.0.0.1:1883"
  localId = "go-local"
  localDataTopic = "channels/local/data"
  localCmdTopic = "channels/local/cmd"
)

var (
  messages chan string
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

func setPublisher(c MQTT.Client, topic string, msg string) {
  var buf bytes.Buffer

  // Build thingspeak data string
  s := strings.Split(msg, ",")
  for i := range s {
    tmp := fmt.Sprintf("field%d=%s", i + 1, s[i])
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

func mqttService() {
  messages = make(chan string)
  cloudCli := mqttClientMaker(cloudUrl, cloudId)
  localCli := mqttClientMaker(localUrl, localId)

  setSubscriber(localCli, localDataTopic, f)

  topic := setTopic(channelId, channelKey)

  for {
    msg := <- messages
    setPublisher(cloudCli, topic, msg)
  }
}
