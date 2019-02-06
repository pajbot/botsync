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
	"time"

	"github.com/gorilla/websocket"
	"github.com/pajlada/botsync/pkg/protocol"
)

var (
	addr = flag.String("addr", "localhost:8080", "http service address")

	errQuit = errors.New("quit lol")

	interrupt = make(chan os.Signal, 1)

	subscribe = make(chan string)
	publish   = make(chan interface{})
	reconnect = make(chan bool)
	doQuit    = make(chan bool)
	input     = make(chan string)

	u url.URL

	text string
)

func connect(u url.URL) error {
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("Received message: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	ticker2 := time.NewTicker(3 * time.Second)
	defer ticker2.Stop()

	for {
		select {
		case <-done:
			return errors.New("xd")
		case msg := <-publish:
			payload, err := json.Marshal(msg)
			err = c.WriteMessage(websocket.TextMessage, payload)
			if err != nil {
				log.Println("write:", err)
				return err
			}
		case topic := <-subscribe:
			msg := &protocol.SubscribeMessage{
				Type:  "SUBSCRIBE",
				Topic: topic,
			}
			payload, err := json.Marshal(msg)
			err = c.WriteMessage(websocket.TextMessage, payload)
			if err != nil {
				log.Println("write:", err)
				return err
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return err
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return errQuit
		}
	}
}

func penis() error {
	go func() {
		err := connect(u)
		if err != nil {
			if err == errQuit {
				log.Println("quitting on demand")
				doQuit <- true
				return
			}

			log.Println("connect error:", err)
			time.AfterFunc(time.Second*1, func() {
				reconnect <- true
			})
		}
	}()

	for {
		select {
		case text := <-input:
			fmt.Println("read:", text)
			switch text {
			case "subscribe":
				subscribe <- "test"
			case "publish":
				publish <- &protocol.PublishMessage{
					Type:  "PUBLISH",
					Topic: "test",
					Data: protocol.PublishMessageData{
						Payload: "lol",
					},
				}
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
