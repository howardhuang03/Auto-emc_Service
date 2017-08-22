package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

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
	timerChan      chan bool
	ti             time.Timer
)

var eclipseHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	var buf bytes.Buffer
	buf.Write(msg.Payload())

	log.Println("Controller received from eclipse: TOPIC: " + msg.Topic() + ", MSG:" + buf.String())

	controllerChan <- buf.String()
}

var localHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	var buf bytes.Buffer
	buf.Write(msg.Payload())

	log.Println("Controller received from local: TOPIC: " + msg.Topic() + ", MSG:" + buf.String())

	responseChan <- buf.String()
}

func publish(c MQTT.Client, target string, broker string, topic string, msg string) {
	if c == nil {
		log.Println("Can't use empty client to send msg: " + msg)
		return
	}

	token := c.Publish(topic, 0, false, msg)
	log.Println(target + " pulished to " + broker + ", TOPIC:" + topic + ", MSG: " + msg)
	token.Wait()
}

func setTimer(target time.Time, now time.Time, dev string, relay string, action string, t timer) {
	log.Println("Set timer: ")
	log.Println(target.String()+", device: "+dev+", Interval:", t.Interval)
	ti := time.NewTimer(target.Sub(now))
	<-ti.C
	log.Println("Timer expired, check dev status first")

	// Check target device status first
	status := fmt.Sprintf("%s,%s,%s", dev, relay, "status")
	controllerChan <- status
	result := <-responseChan

	if strings.Contains(result, "OFF") {
		msg := fmt.Sprintf("%s,%s,%s,%d", dev, relay, action, t.Interval)
		controllerChan <- msg
	} else {
		log.Println("Skip operation since device is already 'ON'")
	}
	timerChan <- true
}

func initTimer() {
	for k, v := range controllerMap {
		now := time.Now()
		date := now
		for i, t := range v.Timer {
			tt, _ := time.ParseInLocation("2006-01-02 15:04:05", date.Format("2006-01-02 ")+t.Time, time.Local)

			// Check next timer
			if now.Before(tt) {
				go setTimer(tt, now, k, "1", "ON", t)
				break
			}

			// Shift to the fist timer at next day once tiemr not found
			if i+1 == len(v.Timer) {
				date = date.AddDate(0, 0, 1)
				tt, _ = time.ParseInLocation("2006-01-02 15:04:05", date.Format("2006-01-02 ")+v.Timer[0].Time, time.Local)
				go setTimer(tt, now, k, "1", "ON", t)
			}
		}
	}
}

func buildController() {
	// Initialize mqtt client
	controllerChan = make(chan string)
	responseChan = make(chan string)
	timerChan = make(chan bool)
	cloudCli := mqttClientMaker(controllerUrl, controllerId)
	localCli := mqttClientMaker(localUrl, controllerId)

	// Subscribe to eclipse mqtt
	setSubscriber(cloudCli, localCmdTopic, eclipseHandler)

	// Subscribe to local mqtt
	setSubscriber(localCli, localResponseTopic, localHandler)

	// Initialize first timer
	initTimer()

	for {
		select {
		case msgC := <-controllerChan:
			publish(localCli, "Controller", "local", localCmdTopic, msgC)
		case msgR := <-responseChan:
			publish(cloudCli, "Controller", "eclipse", localResponseTopic, msgR)
		case <-timerChan:
			initTimer()
		}
	}
}
