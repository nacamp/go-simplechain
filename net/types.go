package net

import (
	"encoding/json"
)

type Message struct {
	Command string
	Data    string
}

func (message *Message) Marshal() []byte {
	m, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	return m
}

func (message *Message) Unmarshal(jsonBytes []byte) {
	err := json.Unmarshal(jsonBytes, message)
	if err != nil {
		panic(err)
	}
}
