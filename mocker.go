package main

import (
	"fmt"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	mockerUrl = localUrl
	mockerId  = "go-mocker"
)

var (
	mockerChan chan string
)

var mockerHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Received: TOPIC: %s, MSG: %s\n", msg.Topic(), msg.Payload())

	mockerChan <- "mocker responsed."
}

func buildMocker() {
	// Initialize mqtt client
	mockerChan = make(chan string)
	localCli := mqttClientMaker(localUrl, mockerId)

	// Subscribe to local mqtt
	setSubscriber(localCli, localCmdTopic, mockerHandler)

	for {
		publish(localCli, localResponseTopic, <-mockerChan)
	}
}
