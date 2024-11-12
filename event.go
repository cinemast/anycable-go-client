package anycable

import (
	"encoding/json"
)

type Event struct {
	Type       string          `json:"type"`
	Message    json.RawMessage `json:"message"`
	Identifier *string         `json:"identifier"`
}

func MessageToStruct[T any](e Event) (*T, error) {
	var r T
	err := json.Unmarshal(e.Message, &r)
	return &r, err
}

func (e Event) GetIdentifier() *ChannelIdentifier {
	if e.Identifier == nil {
		return nil
	}
	id := &ChannelIdentifier{}
	err := json.Unmarshal([]byte(*e.Identifier), id)
	if err != nil {
		return nil
	}
	return id
}

type ChannelIdentifier struct {
	Channel   string `json:"channel"`
	Mode      string `json:"mode"`
	ChannelId string `json:"channel_id"`
	Language  string `json:"language"`
}

func (ci ChannelIdentifier) String() *string {
	txt, err := json.Marshal(ci)
	if err != nil {
		return nil
	}
	res := string(txt)
	return &res
}
