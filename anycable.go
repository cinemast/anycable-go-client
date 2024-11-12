package anycable

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log/slog"
	"sync"
)

type Client struct {
	url           string
	logger        *slog.Logger
	ws            *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	subscriptions map[ChannelIdentifier]*Subscription
	mutex         sync.Mutex
}

type Command struct {
	Command    string  `json:"command"`
	Identifier *string `json:"identifier,omitempty"`
	Data       *string `json:"data,omitempty"`
	Action     *string `json:"action,omitempty"`
}

func NewClient(ctx context.Context, url string, logger *slog.Logger) *Client {
	subCtx, cancel := context.WithCancel(ctx)
	return &Client{url, logger, nil, subCtx, cancel, make(map[ChannelIdentifier]*Subscription), sync.Mutex{}}
}

func (a *Client) Connect() error {
	var err error
	a.ws, _, err = websocket.DefaultDialer.Dial(a.url, nil)
	if err != nil {
		return fmt.Errorf("error connecting to AnyCable server: %w", err)
	}

	msg, err := a.readMessage()
	if err != nil {
		return err
	}

	if msg.Type != "welcome" {
		defer a.Close()
		return fmt.Errorf("unexpected message type: %s", msg.Type)
	}

	go func() {
		for {
			ev, err := a.readMessage()
			if err != nil {
				for _, sub := range a.subscriptions {
					close(sub.Messages)
				}
				break
			}
			switch ev.Type {
			case "ping":
				err := a.sendCommand(Command{Command: "pong"})
				if err != nil {
					a.logger.Error("error writing to AnyCable server", "err", err)
				}
			case "disconnect":
				a.logger.Debug("disconnected from AnyCable server", "ev", ev)
				err = a.Close()
				if err != nil {
					a.logger.Error("error closing AnyCable client", "err", err)
					return
				}
			case "reject_subscription":
				a.mutex.Lock()
				sub, ok := a.subscriptions[*ev.GetIdentifier()]
				a.mutex.Unlock()
				if !ok {
					a.logger.Warn("received reject_subscription for unknown subscription", "identifier", ev.Identifier)
					continue
				}
				close(sub.Messages)
			case "confirm_subscription":
				a.mutex.Lock()
				sub, ok := a.subscriptions[*ev.GetIdentifier()]
				a.mutex.Unlock()
				if !ok {
					a.logger.Warn("received confirm_subscription for unknown subscription", "identifier", ev.Identifier)
					continue
				}
				sub.Subscribed = true
			default:
				if ev.GetIdentifier() == nil {
					a.logger.Warn("received unknown message", "ev", ev)
					continue
				}
				a.mutex.Lock()
				sub, ok := a.subscriptions[*ev.GetIdentifier()]
				a.mutex.Unlock()
				if !ok {
					a.logger.Warn("received message for unknown subscription", "identifier", ev.Identifier)
				}
				sub.Messages <- *ev
			}
		}
	}()
	return nil
}

func (a *Client) Subscribe(identifier ChannelIdentifier) (*Subscription, error) {
	a.mutex.Lock()
	a.subscriptions[identifier] = &Subscription{a, identifier, false, make(chan Event, 1000)}
	a.mutex.Unlock()

	err := a.sendCommand(Command{
		Command:    "subscribe",
		Identifier: identifier.String(),
	})
	if err != nil {
		a.mutex.Lock()
		delete(a.subscriptions, identifier)
		a.mutex.Unlock()
		return nil, err
	}
	return a.subscriptions[identifier], nil
}

func (a *Client) Unsubscribe(subscription *Subscription) error {
	a.mutex.Lock()
	delete(a.subscriptions, subscription.Identifier)
	a.mutex.Unlock()

	return a.sendCommand(Command{
		Command:    "unsubscribe",
		Identifier: subscription.Identifier.String(),
	})
}

func (a *Client) Send(identifier ChannelIdentifier, message any) error {
	return a.sendMessage(identifier, message)
}

func (a *Client) sendMessage(identifier ChannelIdentifier, message any) error {

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error encoding command: %w", err)
	}
	txt := string(data)
	return a.sendCommand(Command{
		Command:    "message",
		Identifier: identifier.String(),
		Data:       &txt,
	})
}

func (a *Client) Close() error {
	a.logger.Debug("anycable: closing connection")
	a.cancel()
	return a.ws.Close()
}

func (a *Client) sendCommand(c Command) error {
	text, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("error encoding command: %w", err)
	}
	return a.sendData(text)
}

func (a *Client) sendData(message []byte) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.logger.Debug("anycable<-" + string(message))
	return a.ws.WriteMessage(websocket.TextMessage, message)
}

func (a *Client) readMessage() (*Event, error) {
	ev := &Event{}

	t, msg, err := a.ws.ReadMessage()
	if err != nil {
		a.logger.Error("error reading message from AnyCable server", "err", err)
		return nil, fmt.Errorf("error reading message from AnyCable server: %w", err)
	}

	if t != websocket.TextMessage {
		return nil, fmt.Errorf("received unexpected message type: %d", t)
	}
	a.logger.Debug("anycable->" + string(msg))
	err = json.Unmarshal(msg, ev)
	if err != nil {
		return nil, fmt.Errorf("could not deserialize message %s: %w", string(msg), err)
	}
	return ev, nil
}
