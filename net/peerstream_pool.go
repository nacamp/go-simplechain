package net

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
)

type PeerStreamHandler interface {
	Register(stream *PeerStream)
	StartHandler()
}

type PeerStreamPool struct {
	mu                   sync.RWMutex //sync.Mutex
	streams              *sync.Map
	handlers             []PeerStreamHandler
	limit                int32
	count                int32
	StatusStreamClosedCh chan interface{}
}

func NewPeerStreamPool() *PeerStreamPool {
	p := PeerStreamPool{streams: new(sync.Map), handlers: make([]PeerStreamHandler, 0)}
	p.limit = 10
	p.StatusStreamClosedCh = make(chan interface{}, 1)
	return &p
}

func (p *PeerStreamPool) SetLimit(maxPeers int) {
	p.limit = int32(maxPeers)
}

//only use at Node.HandleStream, Connect
func (p *PeerStreamPool) AddStream(peerStream *PeerStream) error {
	p.mu.Lock()
	if p.count >= p.limit {
		errors.New("Pool was exceeded max limit")
	}
	p.count++
	p.mu.Unlock()

	p.streams.Store(peerStream.stream.Conn().RemotePeer(), peerStream)
	p.register(peerStream)
	p.startHandler()
	for _, h := range p.handlers {
		h.Register(peerStream)
		h.StartHandler()
	}
	return nil
}

func (p *PeerStreamPool) GetStream(id peer.ID) (*PeerStream, error) {
	v, ok := p.streams.Load(id)
	if ok {
		return v.(*PeerStream), nil
	}
	return nil, errors.New("Not found PeetStream")
}

func (p *PeerStreamPool) RemoveStream(id peer.ID) {
	_, ok := p.streams.Load(id)
	fmt.Println(id.Pretty())
	if ok {
		p.streams.Delete(id)
		atomic.AddInt32(&p.count, -1)
	}

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

func (p *PeerStreamPool) register(peerStream *PeerStream) {
	peerStream.Register(StatusStreamClosed, p.StatusStreamClosedCh)
}

func (p *PeerStreamPool) startHandler() {
	go func() {
		for {
			select {
			case ch := <-p.StatusStreamClosedCh:
				msg := ch.(*Message)
				p.RemoveStream(msg.PeerID)
			}
		}
	}()
}