package net

import (
	"errors"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
)

type PeerStreamHandler interface {
	Register(stream *PeerStream)
	StartHandler()
}

type PeerStreamPool struct {
	streams  *sync.Map
	handlers []PeerStreamHandler
}

func NewPeerStreamPool() *PeerStreamPool {
	p := PeerStreamPool{streams: new(sync.Map), handlers: make([]PeerStreamHandler, 0)}
	return &p
}

func (p *PeerStreamPool) AddStream(peerStream *PeerStream) {
	//TODO:check problem when to add same stream
	//TODO:how to do when exceed pool limit
	p.streams.Store(peerStream.stream.Conn().RemotePeer(), peerStream)
	for _, h := range p.handlers {
		h.Register(peerStream)
		h.StartHandler()
	}
}

func (p *PeerStreamPool) GetStream(id peer.ID) (*PeerStream, error) {
	v, ok := p.streams.Load(id)
	if ok {
		return v.(*PeerStream), nil
	}
	return nil, errors.New("Not found PeetStream")
}

func (p *PeerStreamPool) RemoveStream(peerStream *PeerStream) {
}

func (p *PeerStreamPool) AddHandler(handler PeerStreamHandler) {
	p.handlers = append(p.handlers, handler)
}
