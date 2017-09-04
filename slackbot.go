package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var (
	slackChan chan string
)

func buildSlackBot() {
	time.Sleep(200 * time.Millisecond)
	slackChan = make(chan string)

	api := slack.New(slackConfig.Token)
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)
	api.SetDebug(false)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			//fmt.Print("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello

			case *slack.ConnectedEvent:
				//fmt.Println("Infos:", ev.Info)
				//fmt.Println("Connection counter:", ev.ConnectionCount)
				// Replace #general with your Channel ID
				rtm.SendMessage(rtm.NewOutgoingMessage("Hi, I'm ready to work", slackConfig.ChannelId))
				log.Println("Slackbot initialization done!!")

			case *slack.MessageEvent:
				//fmt.Printf("Message: %v\n", ev)
				s := strings.Split(ev.Msg.Text, " ")
				if ev.Channel == slackConfig.ChannelId && strings.Contains(s[0], slackConfig.BotId) {
					log.Println("Slackbot receive msg: '" + ev.Msg.Text + "' from channel " + ev.Channel)
					controllerChan <- s[1]
				}

			case *slack.PresenceChangeEvent:
				//fmt.Printf("Presence Change: %v\n", ev)

			case *slack.LatencyReport:
				//fmt.Printf("Current latency: %v\n", ev.Value)

			case *slack.RTMError:
				log.Println("Error: " + ev.Error())

			case *slack.InvalidAuthEvent:
				log.Println("Invalid credentials")
				return

			default:

				// Ignore other events..
				// fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		case msg := <-slackChan:
			log.Println("Slackbot send msg: '" + msg + "' to channel " + slackConfig.ChannelId)
			rtm.SendMessage(rtm.NewOutgoingMessage(msg, slackConfig.ChannelId))
		}
	}
}
