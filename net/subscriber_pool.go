package net

import (
	"fmt"
	"sync"
)

type SubsriberPool struct {
	subscribersMap *sync.Map
	messageCh      chan *Message
}

func NewSubsriberPool() *SubsriberPool {
	sp := &SubsriberPool{
		subscribersMap: new(sync.Map),
		messageCh:      make(chan *Message),
	}
	return sp
}

func (sp *SubsriberPool) Register(code uint64, subscriber Subscriber) {
	sp.subscribersMap.Store(code, subscriber)
}

func (sp *SubsriberPool) Deregister() {
}

func (sp *SubsriberPool) Start() {
	go sp.Loop()
}

func (sp *SubsriberPool) handleMessage(message *Message) {
	sp.messageCh <- message
}

func (sp *SubsriberPool) Loop() {
	for {
		message := <-sp.messageCh
		//TODO: v is sync.Map later
		v, ok := sp.subscribersMap.Load(message.Code)
		if ok {
			subscriber := v.(Subscriber)
			fmt.Printf("%v", message)
			subscriber.HandleMessage(message)
		} else {
		}
	}
}
