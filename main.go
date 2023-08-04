package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dghubble/oauth1"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	// Twitter config
	twitterRateLimits := make(map[string]int64)

	// MQTT config setup
	mqttBroker := os.Getenv("MQTT_BROKER")
	mqttClientId := os.Getenv("MQTT_CLIENT_ID")
	mqttTopic := os.Getenv("MQTT_TOPIC")
	mqttUsername := os.Getenv("MQTT_USERNAME")
	mqttPassword := os.Getenv("MQTT_PASSWORD")

	// Setup the MQTT client options
	options := mqtt.NewClientOptions().AddBroker(mqttBroker).SetClientID(mqttClientId)
	options.ConnectRetry = true
	options.AutoReconnect = true

	if mqttUsername != "" {
		options.SetUsername(mqttUsername)
		if mqttPassword != "" {
			options.SetPassword(mqttPassword)
		}
	}

	options.OnConnectionLost = func(c mqtt.Client, e error) {
		log.Println("Connection lost")
	}
	options.OnConnect = func(c mqtt.Client) {
		log.Println("Connected")

		t := c.Subscribe(mqttTopic, 2, nil)
		go func() {
			_ = t.Wait()
			if t.Error() != nil {
				log.Printf("Error subscribing: %s\n", t.Error())
			} else {
				log.Println("Subscribed to:", mqttTopic)
			}
		}()
	}
	options.OnReconnecting = func(_ mqtt.Client, co *mqtt.ClientOptions) {
		log.Println("Attempting to reconnect")
	}
	options.DefaultPublishHandler = func(_ mqtt.Client, m mqtt.Message) {
		log.Printf("Received: %s->%s\n", m.Topic(), m.Payload())

		// Unmarshal the received json into a struct
		var mqttMsg MqttMessage
		err := json.Unmarshal(m.Payload(), &mqttMsg)
		if err != nil {
			log.Fatal(err)
		}

		// If we get a message with no Twitter info, ignore it
		if mqttMsg.TwitterConsumerKey == "" {
			return
		}

		// Check to see if we're rate limited, if so return.
		if reset, ok := twitterRateLimits[mqttMsg.TwitterConsumerKey]; ok {
			if reset >= time.Now().Unix() {
				log.Printf("[Rate Limited] %s: %d", mqttMsg.TwitterConsumerKey, reset)
				return
			}

			delete(twitterRateLimits, mqttMsg.TwitterConsumerKey)
		}

		// Setup Twitter configs
		config := oauth1.NewConfig(mqttMsg.TwitterConsumerKey, mqttMsg.TwitterConsumerSecret)
		token := oauth1.NewToken(mqttMsg.TwitterAccessToken, mqttMsg.TwitterAccessSecret)

		httpClient := config.Client(oauth1.NoContext, token)

		// TODO: Add image handling too
		if mqttMsg.Image == "" {
			resp, err := httpClient.Post("https://api.twitter.com/2/tweets", "application/json",
				bytes.NewBuffer([]byte(fmt.Sprintf(`{"text": "%s"}`, mqttMsg.Message))))
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}

			log.Println("Tweet:", string(body))
			log.Println("Headers:", resp.Header)

			XAppLimit24HourRemainingString := resp.Header.Get("X-App-Limit-24Hour-Remaining")
			XAppLimit24HourResetString := resp.Header.Get("X-App-Limit-24Hour-Reset")

			XAppLimit24HourRemaining, err := strconv.Atoi(XAppLimit24HourRemainingString)
			if err != nil {
				log.Fatal(err)
			}

			XAppLimit24HourReset, err := strconv.ParseInt(XAppLimit24HourResetString, 10, 64)
			if err != nil {
				log.Fatal(err)
			}

			// If we sent too many requests, immediately stop for an hour for this account
			if strings.Contains(string(body), "Too Many Requests") {
				twitterRateLimits[mqttMsg.TwitterConsumerKey] = time.Now().Add(1 * time.Hour).Unix()
			}

			// If we're entirely out of requests for the day, stop until the reset time.
			if XAppLimit24HourRemaining <= 0 {
				twitterRateLimits[mqttMsg.TwitterConsumerKey] = XAppLimit24HourReset
			}
		}
	}

	// Setup the MQTT client with the options we set
	mqttClient := mqtt.NewClient(options)

	// Connect to the MQTT server
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	log.Println("Connected")

	// Block indefinitely until something above errors, or we close out.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig

	log.Println("Signal caught -> Exit")
	mqttClient.Disconnect(1000)
}
