package main

import (
	"bytes"
	"fmt"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	controllerUrl      = "tcp://iot.eclipse.org:1883"
	controllerId       = "go-controller"
	localResponseTopic = "channels/local/response"
)

var (
	controllerChan chan string
	responseChan   chan string
)

var eclipseHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Eclipase Received: TOPIC: %s, MSG: %s\n", msg.Topic(), msg.Payload())

	var buf bytes.Buffer
	buf.Write(msg.Payload())
	controllerChan <- buf.String()
}

var localHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Local Received: TOPIC: %s, MSG: %s\n", msg.Topic(), msg.Payload())

	var buf bytes.Buffer
	buf.Write(msg.Payload())
	responseChan <- buf.String()
}

func publish(c MQTT.Client, topic string, msg string) {
	if c == nil {
		fmt.Printf("Can't use empty client to send msg:%s to cloud\n", msg)
		return
	}

	// Publish message
	token := c.Publish(topic, 0, false, msg)

	fmt.Printf("Pulished: TOPIC: %s, MSG: %s\n", topic, msg)
	token.Wait()
}

func buildController() {
	// Initialize mqtt client
	controllerChan = make(chan string)
	responseChan = make(chan string)
	cloudCli := mqttClientMaker(controllerUrl, controllerId)
	localCli := mqttClientMaker(localUrl, controllerId)

	// Subscribe to eclipse mqtt
	setSubscriber(cloudCli, localCmdTopic, eclipseHandler)

	// Subscribe to local mqtt
	setSubscriber(localCli, localResponseTopic, localHandler)

	for {
		publish(localCli, localCmdTopic, <-controllerChan)
		publish(cloudCli, localResponseTopic, <-responseChan)
	}
}
