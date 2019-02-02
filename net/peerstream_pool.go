package net

import (
	"sync"
)

type PeerStreamHandler interface {
	Register(stream *PeerStream) error
}

type PeerStreamPool struct {
	streams  *sync.Map
	handlers []PeerStreamHandler
}

func NewPeerStreamPool() *PeerStreamPool {
	p := PeerStreamPool{}
	return &p
}
func (p *PeerStreamPool) AddStream(peerStream *PeerStream) {
	p.streams.Store(peerStream.stream.Conn().RemotePeer(), peerStream)
	for _, h := range p.handlers {
		h.Register(peerStream)
	}
}

func (p *PeerStreamPool) RemoveStream(peerStream *PeerStream) {
}

func (p *PeerStreamPool) AddHandler(handler PeerStreamHandler) {
	p.handlers = append(p.handlers, handler)
}
