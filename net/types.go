package net

import (
	"github.com/najimmy/go-simplechain/rlp"
)

const (
	MSG_HELLO     = 0x00
	MSG_HELLO_ACK = 0x01
	MSG_PEERS     = 0x02
	MSG_PEERS_ACK = 0x03

	MSG_NEW_BLOCK         = 0x10
	MSG_MISSING_BLOCK     = 0x12
	MSG_MISSING_BLOCK_ACK = 0x13
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
