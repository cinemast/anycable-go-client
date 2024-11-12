package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cinemast/anycable-go-client/pkg/anycable"
	"log/slog"
)

type SampleChannelIdentifier struct {
	Channel string `json:"channel"`
	Mode    string `json:"mode"`
}

type Message struct {
	SomeKey string `json:"some_key"`
}

func (i SampleChannelIdentifier) String() string {
	data, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func main() {

	slog.SetLogLoggerLevel(slog.LevelDebug)
	cl := anycable.NewClient(context.TODO(), "wss://someurl", slog.Default())

	err := cl.Connect()
	if err != nil {
		panic(err)
	}

	channelId := SampleChannelIdentifier{
		Channel: "some channel",
		Mode:    "some mode",
	}

	// Subscribing to messages
	sub, err := cl.Subscribe(channelId)
	if err != nil {
		panic(err)
	}

	// Receiving messages
	for ev := range sub.Messages {
		fmt.Println(anycable.MessageToStruct[Message](ev))
	}

	//Publishing message
	err = cl.Send(channelId, &Message{SomeKey: "bar"})
	if err != nil {
		panic(err)
	}

	// Closing
	err = cl.Close()
	if err != nil {
		panic(err)
	}

}
