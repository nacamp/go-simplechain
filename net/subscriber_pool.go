package net

import (
	"sync"
)

type SubscriberPool struct {
	subscribersMap *sync.Map
	messageCh      chan *Message
}

func NewSubsriberPool() *SubscriberPool {
	sp := &SubscriberPool{
		subscribersMap: new(sync.Map),
		messageCh:      make(chan *Message),
	}
	return sp
}

func (sp *SubscriberPool) Register(code uint64, subscriber Subscriber) {
	sp.subscribersMap.Store(code, subscriber)
}

func (sp *SubscriberPool) Deregister() {
}

func (sp *SubscriberPool) Start() {
	go sp.Loop()
}

func (sp *SubscriberPool) handleMessage(message *Message) {
	sp.messageCh <- message
}

func (sp *SubscriberPool) Loop() {
	for {
		message := <-sp.messageCh
		//TODO: v is sync.Map later
		v, ok := sp.subscribersMap.Load(message.Code)
		if ok {
			subscriber := v.(Subscriber)
			// log.CLog().Info("%v", message)
			subscriber.HandleMessage(message)
		} else {
		}
	}
}
