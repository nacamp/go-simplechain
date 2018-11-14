package net

import (
	"github.com/najimmy/go-simplechain/rlp"
)

const (
	CMD_HELLO     = 0x00
	CMD_HELLO_ACK = 0x01
	CMD_PEERS     = 0x02
	CMD_PEERS_ACK = 0x03

	CMD_BLOCK = 0x10
)

type Message struct {
	Code    uint64
	Payload []byte
}

func NewRLPMessage(code uint64, payload interface{}) (msg Message, err error) {
	msg.Code = code
	if encodedBytes, err := rlp.EncodeToBytes(payload); err != nil {
		return msg, err
	} else {
		msg.Payload = encodedBytes
	}
	return msg, nil
}

type Subscriber interface {
	HandleMessage(message *Message) error
}
