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
	timerChan      chan string
	ti             time.Timer
)

var eclipseHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	var buf bytes.Buffer
	buf.Write(msg.Payload())

	log.Println("Controller received from eclipse, TOPIC: " + msg.Topic() + ", MSG: " + buf.String())

	controllerChan <- buf.String()
}

var localHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	var buf bytes.Buffer
	buf.Write(msg.Payload())

	log.Println("Controller received from local, TOPIC: " + msg.Topic() + ", MSG: " + buf.String())

	responseChan <- buf.String()
}

func publish(c MQTT.Client, target string, broker string, topic string, msg string) {
	if c == nil {
		log.Println("Can't use empty client to send msg: " + msg)
		return
	}

	token := c.Publish(topic, 0, false, msg)
	log.Println(target + " pulished to " + broker + ", TOPIC: " + topic + ", MSG: " + msg)
	token.Wait()
}

func setTimer(target time.Time, now time.Time, dev string, relay string, action string, t timer) {
	log.Println("Set timer: ")
	log.Println(target.String() + ", device: " + dev + ", relay:" + relay + ", Interval: " + fmt.Sprintf("%d", t.Interval) + " mins")
	ti := time.NewTimer(target.Sub(now))
	<-ti.C
	log.Println("Timer expired, check dev status first")

	// Check target device status first
	status := fmt.Sprintf("%s,%s,%s", dev, relay, "status")
	controllerChan <- status
	result := <-responseChan

	if strings.Contains(result, dev) && strings.Contains(result, relay) && strings.Contains(result, "OFF") {
		msg := fmt.Sprintf("%s,%s,%s,%d", dev, relay, action, t.Interval)
		controllerChan <- msg
	} else {
		log.Println("Skip operation since device is already 'ON'")
	}
	timerChan <- fmt.Sprintf("%s,%s", dev, relay)
}

func checkTimer(s string) {
	for k, c := range controllerMap {
		now := time.Now()
		date := now
		dev := k
		for i, r := range c.Relay {
			relay := fmt.Sprint(i + 1)

			// Check which timer should be enabled
			// Init means to initialize all timer
			ss := fmt.Sprintf("%s,%s", dev, relay)
			if s != "init" && s != ss {
				continue
			}

			// Set correct timer
			for j, t := range r.Timer {
				tt, _ := time.ParseInLocation("2006-01-02 15:04:05", date.Format("2006-01-02 ")+t.Time, time.Local)

				// Check next timer
				if now.Before(tt) {
					go setTimer(tt, now, dev, relay, "ON", t)
					break
				}

				// Shift to the fist timer at next day once tiemr not found
				if j+1 == len(r.Timer) {
					date = date.AddDate(0, 0, 1)
					tt, _ = time.ParseInLocation("2006-01-02 15:04:05", date.Format("2006-01-02 ")+r.Timer[0].Time, time.Local)
					go setTimer(tt, now, k, "1", "ON", t)
				}
			}
		}
	}
}

func buildController() {
	// Initialize mqtt client
	controllerChan = make(chan string)
	responseChan = make(chan string)
	timerChan = make(chan string)
	cloudCli := mqttClientMaker(controllerUrl, controllerId, eclipseHandler)
	localCli := mqttClientMaker(localUrl, controllerId, localHandler)
	ticker := time.NewTicker(time.Minute * 10)

	// Subscribe to eclipse mqtt
	setSubscriber(cloudCli, localCmdTopic, eclipseHandler)

	// Subscribe to local mqtt
	setSubscriber(localCli, localResponseTopic, localHandler)

	// Initialize first timer
	checkTimer("init")

	for {
		select {
		case msgC := <-controllerChan:
			publish(localCli, "Controller", "local", localCmdTopic, msgC)
		case msgR := <-responseChan:
			publish(cloudCli, "Controller", "eclipse", localResponseTopic, msgR)
		case s := <-timerChan:
			checkTimer(s)
		case <-ticker.C:
			log.Println("Subscribe eclipse topic: " + localCmdTopic)
			setSubscriber(cloudCli, localCmdTopic, eclipseHandler)
		}
	}
}
