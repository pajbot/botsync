package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/pajlada/botsync/pkg/client"
	"github.com/pajlada/botsync/pkg/protocol"
)

var (
	addr = flag.String("addr", "localhost:8080", "http service address")

	errQuit = errors.New("quit lol")

	interrupt = make(chan os.Signal, 1)

	reconnect = make(chan bool)
	doQuit    = make(chan bool)
	input     = make(chan string)

	u url.URL

	text string
)

func penis() error {
	c := client.NewClient(u.String())
	c.SetAuthentication(protocol.Authentication{
		TwitchUserID:        "82008718",
		AuthenticationToken: "penis",
	})
	c.OnMessage(func(message *protocol.Message) {
		switch message.Topic {
		case "afk":
			parameters := &protocol.AFKParameters{}
			json.Unmarshal(message.Data, parameters)
			fmt.Println("Received AFK update:", parameters)
		default:
			fmt.Println("Got message for unhandled topic:", message.Topic)
		}
	})
	go func() {
		err := c.Connect()
		fmt.Println("connect ended:", err)
		doQuit <- true
	}()

	for {
		select {
		case <-interrupt:
			c.Disconnect()
		case text := <-input:
			parts := strings.Split(text, " ")
			switch parts[0] {
			case "afksubscribe":
				channelID := "11148817"
				if len(parts) >= 2 {
					channelID = parts[1]
				}
				c.Send(protocol.NewAFKSubscribeMessage(channelID))
			case "publish":
				// publish <- &protocol.PublishMessage{
				// 	Type:  "PUBLISH",
				// 	Topic: "test",
				// 	Data: protocol.PublishMessageData{
				// 		Payload: "lol",
				// 	},
				// }
			case "ping":
				c.Send(protocol.NewPingMessage())
			case "afk":
				if len(parts) < 2 {
					log.Println("Unsufficient arguments to afk")
					continue
				}
				var reason string
				username := parts[1]
				if len(parts) >= 3 {
					reason = strings.Join(parts[2:], " ")
				}
				c.Send(protocol.NewAFKMessage(&protocol.AFKParameters{
					UserID:   "40286300",
					UserName: username,

					ChannelID:   "11148817",
					ChannelName: "pajlada",

					Reason: reason,
				}))
			default:
				log.Println("unhandled text:", text)
			}
		case <-doQuit:
			return errQuit
		case <-reconnect:
			return nil
		}
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	signal.Notify(interrupt, os.Interrupt)

	u = url.URL{Scheme: "ws", Host: *addr, Path: "/ws/pubsub"}
	fmt.Println("Host:", u.String())

	reader := bufio.NewReader(os.Stdin)

	go func() {
		for {
			fmt.Print("Type command: ")
			text, _ = reader.ReadString('\n')
			input <- strings.TrimSpace(text)
		}
	}()

	for {
		err := penis()
		if err == errQuit {
			return
		}
	}
}
