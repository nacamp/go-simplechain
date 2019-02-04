package net

import (
	"errors"
	"math/rand"
	"sync"
	"time"

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

//only use at Node.HandleStream, Connect
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

func (p *PeerStreamPool) RemoveStream(id peer.ID) {
	p.streams.Delete(id)
}

func (p *PeerStreamPool) AddHandler(handler PeerStreamHandler) {
	p.handlers = append(p.handlers, handler)
}

func (p *PeerStreamPool) SendMessageToRandomNode(message *Message) error {
	ids := make([]peer.ID, 0)
	p.streams.Range(func(key, value interface{}) bool {
		id := key.(peer.ID)
		ids = append(ids, id)
		return true
	})
	size := len(ids)
	if size == 0 {
		return errors.New("Not found peerinfo")
	}
	rand.Seed(time.Now().Unix())
	id := ids[rand.Intn(len(ids))]
	ps, err := p.GetStream(id)
	if err != nil {
		return err
	}
	ps.SendMessage(message)
	return nil
}

func (p *PeerStreamPool) BroadcastMessage(message *Message) {
	p.streams.Range(func(key, value interface{}) bool {
		ps := value.(*PeerStream)
		ps.SendMessage(message)
		return true
	})
}
