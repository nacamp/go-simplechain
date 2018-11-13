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
	Code    uint64
	Payload []byte
}

func NewRLPMessage(code uint64, payload interface{}) (msg Message) {
	msg.Code = code
	encodedBytes, _ := rlp.EncodeToBytes(payload)
	msg.Payload = encodedBytes
	return msg
}
