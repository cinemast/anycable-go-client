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
