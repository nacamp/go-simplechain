package net

import (
	"github.com/najimmy/go-simplechain/rlp"
)

const (
	CMD_HELLO     = 0x00
	CMD_HELLO_ACK = 0x01
	CMD_PEERS     = 0x02
	CMD_PEERS_ACK = 0x03
)

type Message struct {
	Code uint64
	// Payload interface{}
	Payload []byte
}

func NewMessageEncodePayload(code uint64, payload interface{}) (msg Message) {
	encodedBytes, _ := rlp.EncodeToBytes(payload)
	msg.Payload = encodedBytes
	msg.Code = code
	// rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&msg)
	return msg
}

func (m *Message) Decode(payload *string) {
	rlp.DecodeBytes(m.Payload, payload)
}
