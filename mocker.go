package main

import (
	"bytes"
	"log"

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
	var buf bytes.Buffer
	buf.Write(msg.Payload())

	log.Println("Mocker received from local: TOPIC:", msg.Topic(), " MSG:", buf.String())

	mockerChan <- "mocker responsed."
}

func buildMocker() {
	// Initialize mqtt client
	mockerChan = make(chan string)
	localCli := mqttClientMaker(localUrl, mockerId, mockerHandler)

	// Subscribe to local mqtt
	setSubscriber(localCli, localCmdTopic, mockerHandler)

	for {
		publish(localCli, "Mocker", "local", localResponseTopic, <-mockerChan)
	}
}
