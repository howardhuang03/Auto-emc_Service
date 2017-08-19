package main

import (
	"bytes"
	"log"

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
	var buf bytes.Buffer
	buf.Write(msg.Payload())

	log.Println("Controller received from eclipse: TOPIC:", msg.Topic(), "MSG:", buf.String())

	controllerChan <- buf.String()
}

var localHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	var buf bytes.Buffer
	buf.Write(msg.Payload())

	log.Println("Controller received from local: TOPIC:", msg.Topic(), "MSG:", buf.String())

	responseChan <- buf.String()
}

func publish(c MQTT.Client, target string, broker string, topic string, msg string) {
	if c == nil {
		log.Println("Can't use empty client to send msg:", msg)
		return
	}

	token := c.Publish(topic, 0, false, msg)
	log.Println(target, "pulished to", broker, ", TOPIC:", topic, "MSG:", msg)
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
		publish(localCli, "Controller", "Eclipse", localCmdTopic, <-controllerChan)
		publish(cloudCli, "Controller", "Local", localResponseTopic, <-responseChan)
	}
}
