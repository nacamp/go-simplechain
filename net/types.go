package net

import (
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/nacamp/go-simplechain/rlp"
)

const (
	MsgHello           = uint64(0x00)
	MsgHelloAck        = uint64(0x01)
	MsgNearestPeers    = uint64(0x04)
	MsgNearestPeersAck = uint64(0x05)

	MsgNewBlock         = 0x10
	MsgMissingBlock     = 0x12
	MsgMissingBlockAck  = 0x13
	MsgMissingBlocks    = 0x14
	MsgMissingBlocksAck = 0x15
	MsgNewTx            = 0x16

	StatusStreamClosed = 0x101
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
