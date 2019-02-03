package net

import (
	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/nacamp/go-simplechain/rlp"
)

const (
	MsgHello           = uint64(0x00)
	MsgHelloAck        = uint64(0x01)
	MSG_PEERS          = 0x02
	MSG_PEERS_ACK      = 0x03
	MsgNearestPeers    = uint64(0x04)
	MsgNearestPeersAck = uint64(0x05)

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

type PeerInfo2 struct {
	ID   peer.ID
	Addr []byte
}

func ToPeerInfo2(info *peerstore.PeerInfo) *PeerInfo2 {
	for _, addr := range info.Addrs {
		//why p2p-circuit ?
		if addr.String() != "/p2p-circuit" {
			return &PeerInfo2{ID: info.ID, Addr: addr.Bytes()}
		}
	}
	return nil

}

func FromPeerInfo2(info *PeerInfo2) *peerstore.PeerInfo {
	addr, _ := ma.NewMultiaddrBytes(info.Addr)
	return &peerstore.PeerInfo{ID: info.ID, Addrs: []ma.Multiaddr{addr}}
}

type IConnect interface {
	Connect(id peer.ID, addr ma.Multiaddr) (*PeerStream, error)
}
