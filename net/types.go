package net

import (
	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/nacamp/go-simplechain/rlp"
)

const (
	MSG_HELLO     = 0x00
	MSG_HELLO_ACK = 0x01
	MSG_PEERS     = 0x02
	MSG_PEERS_ACK = 0x03

	MSG_NEW_BLOCK         = 0x10
	MSG_MISSING_BLOCK     = 0x12
	MSG_MISSING_BLOCK_ACK = 0x13
	MSG_NEW_TX            = 0x14
)

type Message struct {
	Code    uint64
	PeerID  peer.ID
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

type INode interface {
	RegisterSubscriber(code uint64, subscriber Subscriber)
	HandleStream(s libnet.Stream)
	SendMessage(message *Message, peerID peer.ID)
	SendMessageToRandomNode(message *Message)
	BroadcastMessage(message *Message)
}
