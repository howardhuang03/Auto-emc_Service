package main

import (
  "os"
  "fmt"
  "time"
  "bytes"
  "strings"

  MQTT "github.com/eclipse/paho.mqtt.golang"
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

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
  fmt.Printf("TOPIC: %s\n", msg.Topic())
  fmt.Printf("MSG: %s\n", msg.Payload())

  // Write incoming data to file
  var buffer bytes.Buffer
  buffer.Write(msg.Payload())
  buffer.WriteString("\n")
  n3, err := localFile.Write(buffer.Bytes())
  _ = n3
  check(err)
  localFile.Sync()
}

func mqttLocalHandler() {
  //create a ClientOptions struct setting the broker address, clientid, turn
  //off trace output and set the default message handler
  opts := MQTT.NewClientOptions().AddBroker("tcp://127.0.0.1:1883")
  opts.SetClientID("go-local")
  opts.SetDefaultPublishHandler(f)

  //create and start a client using the above ClientOptions
  c := MQTT.NewClient(opts)
  if token := c.Connect(); token.Wait() && token.Error() != nil {
    panic(token.Error())
  }

  //subscribe to the topic /go-mqtt/sample and request messages to be delivered
  //at a maximum qos of zero, wait for the receipt to confirm the subscription
  if token := c.Subscribe("topic99", 0, nil); token.Wait() && token.Error() != nil {
    fmt.Println(token.Error())
    os.Exit(1)
  }
}

func mqttCloudHandler() {
  id := "XXX"
  key := "XXX"
  topic := fmt.Sprintf("channels/%s/publish/%s", id, key)

  opts := MQTT.NewClientOptions().AddBroker("tcp://mqtt.thingspeak.com:1883")
  opts.SetClientID("go-cloud")

  c := MQTT.NewClient(opts)
  if token := c.Connect(); token.Wait() && token.Error() != nil {
    panic(token.Error())
  }

  //Publish 5 messages to ThingSpeak at qos 0 and wait for the receipt
  //from the server after sending each message
  for i := 0; i < 5; i++ {
    text := fmt.Sprintf("field1=%d&field2=%d", i, i + 10)
    token := c.Publish(topic, 0, false, text)
    fmt.Printf("TOPIC: %s, MSG: %s\n", topic, text)
    token.Wait()
    time.Sleep(20 * time.Second)
  }
}
