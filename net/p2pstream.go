package net

import (
	"bufio"
	"context"
	"fmt"
	"sync"

	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/sirupsen/logrus"
)

type P2PStream struct {
	mu                  sync.RWMutex //sync.Mutex
	peerID              peer.ID
	addr                ma.Multiaddr
	stream              libnet.Stream
	node                *Node
	isFinishedHandshake bool
	isClosed            bool
	finshedHandshakeCh  chan bool
	messageCh           chan *Message
}

func NewP2PStream(node *Node, peerID peer.ID) (*P2PStream, error) {
	s, err := node.host.NewStream(context.Background(), peerID, "/simplechain/0.0.1")
	if err != nil {
		log.CLog().Warning(err)
		return nil, err
	}
	P2PStream := &P2PStream{
		node:               node,
		stream:             s,
		peerID:             peerID,
		addr:               s.Conn().RemoteMultiaddr(),
		finshedHandshakeCh: make(chan bool),
		messageCh:          make(chan *Message, 100),
	}
	return P2PStream, nil
}

func NewP2PStreamWithStream(node *Node, s libnet.Stream) (*P2PStream, error) {
	fmt.Println(s.Conn().RemoteMultiaddr())
	P2PStream := &P2PStream{
		node:               node,
		stream:             s,
		peerID:             s.Conn().RemotePeer(),
		addr:               s.Conn().RemoteMultiaddr(),
		finshedHandshakeCh: make(chan bool),
		messageCh:          make(chan *Message),
	}
	return P2PStream, nil
}

func (ps *P2PStream) Start(isHost bool) {
	log.CLog().Info("Start")
	rw := bufio.NewReadWriter(bufio.NewReader(ps.stream), bufio.NewWriter(ps.stream))
	go ps.readData(rw)
	go ps.writeData(rw)
	if !isHost {
		ps.SendHello()
	}

}

func (ps *P2PStream) readData(rw *bufio.ReadWriter) {
	for {
		message := Message{}
		err := rlp.Decode(rw, &message)
		if err != nil {
			//time.Sleep(30 * time.Second)
			ps.stream.Close()
			log.CLog().Debug("readData  lock before")
			ps.mu.Lock()
			ps.isClosed = true
			ps.mu.Unlock()
			log.CLog().Debug("readData  Unlock after")
			ps.node.host.Peerstore().ClearAddrs(ps.peerID)
			//P2PStream.node.host.Peerstore().AddAddr(P2PStream.peerID, P2PStream.addr, 0)
			log.CLog().Debug("client closed")
			return
		}
		switch message.Code {
		case CMD_HELLO:
			ps.onHello(&message)
		case CMD_HELLO_ACK:
			ps.onHelloAck(&message)
		default:
			fmt.Println("lock...")
			if !ps.isFinishedHandshake {
				continue
			}
			fmt.Println("unlock...")
		}
		switch message.Code {
		case CMD_PEERS:
			ps.onPeers(&message)
		case CMD_PEERS_ACK:
			ps.onPeersAck(&message)
		default:
			//subscribe
			ps.node.subsriberPool.handleMessage(&message)
		}
	}
}

func (ps *P2PStream) writeData(rw *bufio.ReadWriter) {
	<-ps.finshedHandshakeCh
	for {
		select {
		case message := <-ps.messageCh:
			ps.sendMessage(message)
			// continue
		default:
		}
	}
}

func (ps *P2PStream) SendHello() error {
	if msg, err := NewRLPMessage(CMD_HELLO, ps.node.maddr.String()); err != nil {
		return err
	} else {
		log.CLog().Info("SendHello")
		return ps.sendMessage(&msg)
	}
}

func (ps *P2PStream) SendHelloAck() error {
	if msg, err := NewRLPMessage(CMD_HELLO_ACK, ps.node.maddr.String()); err != nil {
		return err
	} else {
		log.CLog().Info("SendHelloAck")
		return ps.sendMessage(&msg)
	}
}

func (ps *P2PStream) onHello(message *Message) error {
	defer ps.finshHandshake()
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.CLog().WithFields(logrus.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Info("onHello")

	node := ps.node
	addr, err := ma.NewMultiaddr(data)
	if err != nil {
		return err
	}
	node.nodeRoute.Update(ps.peerID, addr) //P2PStream.addr
	return ps.SendHelloAck()
}

func (ps *P2PStream) onHelloAck(message *Message) error {
	defer ps.finshHandshake()
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.CLog().WithFields(logrus.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Info("onHelloAck")

	node := ps.node
	addr, err := ma.NewMultiaddr(data)
	if err != nil {
		return err
	}
	node.nodeRoute.Update(ps.peerID, addr) //P2PStream.addr
	return nil
}

//send request peers
func (ps *P2PStream) RequestPeers() error {
	if msg, err := NewRLPMessage(CMD_PEERS, "version 0.1"); err != nil {
		return err
	} else {
		log.CLog().Info("RequestPeers")
		ps.messageCh <- &msg
	}
	return nil
}

func (ps *P2PStream) RequestPeersAck() error {
	log.CLog().Info("RequestPeersAck>>>>>")
	node := ps.node

	peers := node.nodeRoute.NearestPeers(ps.peerID, 10)
	payload := make([][]string, 0)
	for k, addr := range peers {
		payload = append(payload, []string{k.Pretty(), addr.String()})
	}

	if msg, err := NewRLPMessage(CMD_PEERS_ACK, &payload); err != nil {
		return err
	} else {
		log.CLog().Info("<<<<<RequestPeersAck")
		ps.messageCh <- &msg
	}
	return nil
}

func (ps *P2PStream) onPeers(message *Message) error {
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.CLog().WithFields(logrus.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Info("onPeers")
	return ps.RequestPeersAck()
}

func (ps *P2PStream) onPeersAck(message *Message) error {
	log.CLog().Info("onPeersAck>>>>>")
	payload := make([][]string, 0)

	err := rlp.DecodeBytes(message.Payload, &payload)
	if err != nil {
		log.CLog().Warning(err)
		return err
	}

	node := ps.node
	for _, addr := range payload {
		fmt.Printf("%v\n", addr)
		id, _ := peer.IDB58Decode(addr[0])
		maddr, _ := ma.NewMultiaddr(addr[1])
		node.nodeRoute.Update(id, maddr)
	}

	node.nodeRoute.NearestPeers(node.host.ID(), 10)
	log.CLog().Info("<<<<<onPeersAck")
	return nil
}

func (ps *P2PStream) sendMessage(message *Message) error {
	encodedBytes, _ := rlp.EncodeToBytes(message)
	_, err := ps.stream.Write(encodedBytes)
	if err != nil {
		ps.stream.Close()
		log.CLog().Debug("sendMessage lock before")
		ps.mu.Lock()
		ps.isClosed = true
		ps.mu.Unlock()
		log.CLog().Debug("sendMessage Unlock after")
		ps.node.host.Peerstore().ClearAddrs(ps.peerID)
		//ps.node.host.Peerstore().AddAddr(ps.peerID, ps.addr, 0)
		log.CLog().Warning("sendMessage: client closed")
		return err
	}
	return nil
}

func (ps *P2PStream) finshHandshake() {
	log.CLog().Debug("finshHandshake lock before")
	ps.finshedHandshakeCh <- true
	ps.mu.Lock()
	ps.isFinishedHandshake = true
	ps.mu.Unlock()
	log.CLog().Debug("finshHandshake Unlock after")
	log.CLog().Info("finshHandshake")
}
