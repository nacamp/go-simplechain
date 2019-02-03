package net

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"time"

	ma "github.com/multiformats/go-multiaddr"

	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/sirupsen/logrus"
)

const (
	ConcurrencyLimit = 3
	BucketSize       = 16
)

type Discovery struct {
	peerstore    peerstore.Peerstore
	routingTable *kb.RoutingTable
	_findnode    func(peerInfo *peerstore.PeerInfo, targetID peer.ID) []*peerstore.PeerInfo
	_bond        func(peerInfo *peerstore.PeerInfo) *peerstore.PeerInfo

	// MsgHelloCh    chan interface{}
	HandshakeSucceedCh   chan interface{}
	MsgNearestPeersCh    chan interface{}
	MsgNearestPeersAckCh chan interface{}
	conn                 IConnect
	streamPool           *PeerStreamPool
	hostAddr             ma.Multiaddr
	hostID               peer.ID
}

func NewDiscovery(hostID peer.ID, hostAddr ma.Multiaddr, metrics peerstore.Metrics, peerstore peerstore.Peerstore, streamPool *PeerStreamPool, conn IConnect) *Discovery {
	d := &Discovery{hostID: hostID, hostAddr: hostAddr, streamPool: streamPool, conn: conn}
	d.routingTable =
		kb.NewRoutingTable(BucketSize, kb.ConvertPeerID(hostID), time.Minute, metrics)
	d.peerstore = peerstore
	d.MsgNearestPeersCh = make(chan interface{}, 1)
	d.MsgNearestPeersAckCh = make(chan interface{}, 1)
	d.HandshakeSucceedCh = make(chan interface{}, 1)
	return d
}

func (d *Discovery) Update(peerInfo *peerstore.PeerInfo) {
	d.routingTable.Update(peerInfo.ID)
	d.peerstore.AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Duration(3600)*time.Second)
}

func (d *Discovery) UpdateAddr(id peer.ID, addr ma.Multiaddr) {
	d.routingTable.Update(id)
	d.peerstore.AddAddr(id, addr, time.Duration(3600)*time.Second)
}

func (d *Discovery) RandomPeerInfo() (peerstore.PeerInfo, error) {
	ids := d.peerstore.Peers()
	size := len(ids)
	if size == 0 {
		return peerstore.PeerInfo{}, errors.New("Not found peerinfo")
	}
	rand.Seed(time.Now().Unix())
	id := ids[rand.Intn(len(ids))]
	return d.peerstore.PeerInfo(id), nil
}

func sortByDistance(peerInfos []*peerstore.PeerInfo, targetID peer.ID) []*peerstore.PeerInfo {
	IDs := make([]peer.ID, 0, len(peerInfos))
	infos := make(map[peer.ID]*peerstore.PeerInfo)
	for i, id := range peerInfos {
		IDs = append(IDs, id.ID)
		infos[id.ID] = peerInfos[i]
	}

	peers := kb.SortClosestPeers(IDs, kb.ConvertPeerID(targetID))
	closestSize := ConcurrencyLimit
	if len(peers) < ConcurrencyLimit {
		closestSize = len(peers)
	}
	closet := make([]*peerstore.PeerInfo, 0, closestSize)
	for _, ID := range peers[:closestSize] {
		closet = append(closet, infos[ID])
	}
	return closet
}

func (d *Discovery) NearestPeers(peerID peer.ID) []*peerstore.PeerInfo {
	closest := d.routingTable.NearestPeers(kb.ConvertPeerID(peerID), BucketSize)
	closestPeerInfo := make([]*peerstore.PeerInfo, 0, BucketSize)
	for _, id := range closest {
		p := d.peerstore.PeerInfo(id)
		closestPeerInfo = append(closestPeerInfo, &p)
	}
	return closestPeerInfo
}

func (d *Discovery) findnode(peerInfo *peerstore.PeerInfo, targetID peer.ID, reply chan<- []*peerstore.PeerInfo) {
	if d._findnode != nil {
		reply <- d._findnode(peerInfo, targetID)
		return
	}
	//TODO: error
	peerStream, _ := d.bond(peerInfo)

	msg, _ := NewRLPMessage(MsgNearestPeers, targetID)
	replyAck := make(chan interface{}, 1)
	peerStream.SendMessageReply(&msg, replyAck)
	ack := <-replyAck
	reply <- ack.([]*peerstore.PeerInfo)
}

func (d *Discovery) bond(peerInfo *peerstore.PeerInfo) (*PeerStream, error) {
	if d._bond != nil {
		d._bond(peerInfo)
		return nil, nil
		// reply <- d._bond(peerInfo)
		// d.Update(peerInfo)
		// return nil, nil
	}
	peerStream, err := d.conn.Connect(peerInfo.ID, peerInfo.Addrs[0])
	if err != nil {
		return nil, err
	}
	//TODO: make func
	if peerStream.status != statusHandshakeSucceed {
		peerStream.SendHello(d.hostAddr)
		//TODO: timeout
		<-peerStream.HandshakeSucceedCh
	} else {
		//TODO: stream status check
	}
	d.Update(peerInfo)
	d.streamPool.AddStream(peerStream)
	return peerStream, nil
}

func (d *Discovery) bondReply(peerInfo *peerstore.PeerInfo, reply chan<- *peerstore.PeerInfo) error {
	if d._bond != nil {
		reply <- d._bond(peerInfo)
		d.Update(peerInfo)
		return nil
	}
	_, err := d.bond(peerInfo)
	if err != nil {
		return err
	}
	reply <- peerInfo
	return nil
}

func (d *Discovery) Register(peerStream *PeerStream) {
	fmt.Println("register")
	peerStream.Register(MsgNearestPeers, d.MsgNearestPeersCh)
	peerStream.Register(MsgNearestPeersAck, d.MsgNearestPeersAckCh)
	peerStream.Register(MsgHello, d.HandshakeSucceedCh)
}

func (d *Discovery) StartHandler() {
	fmt.Println("StartHandler")
	go d.handleMsgNearestPeers()
	go d.handleMsgNearestPeersAck()
	go d.handleMsgHello()
}

func (d *Discovery) handleMsgHello() {
	for {
		select {
		case ch := <-d.HandshakeSucceedCh:
			message := ch.(*Message)
			data := string("")
			rlp.DecodeBytes(message.Payload, &data)
			log.CLog().WithFields(logrus.Fields{
				"ID": message.PeerID,
			}).Debug("addr: ", data)
			addr, err := ma.NewMultiaddr(data)
			if err != nil {
				continue
			}
			d.UpdateAddr(message.PeerID, addr)
		}
	}
}

func (d *Discovery) SendNearestPeers(targetID peer.ID, ps *PeerStream) error {
	closestPeerInfo := d.NearestPeers(targetID)
	payload := make([]*PeerInfo2, 0)
	for _, info := range closestPeerInfo {
		// p := d.peerstore.PeerInfo(id)
		payload = append(payload, ToPeerInfo2(info))
	}
	if msg, err := NewRLPMessage(MsgNearestPeersAck, &payload); err != nil {
		return err
	} else {
		ps.SendMessage(&msg)
	}
	return nil
}

func (d *Discovery) handleMsgNearestPeers() {
	for {
		select {
		case ch := <-d.MsgNearestPeersCh:
			// fmt.Println(msg)
			msg := ch.(*Message)
			fmt.Println(msg.Code)
			var targetID peer.ID
			_ = rlp.DecodeBytes(msg.Payload, &targetID)
			ps, _ := d.streamPool.GetStream(msg.PeerID)
			d.SendNearestPeers(targetID, ps)
			log.CLog().WithFields(logrus.Fields{}).Debug("targetID: ", targetID)
		}
	}
}

func (d *Discovery) handleMsgNearestPeersAck() {
	for {
		select {
		case ch := <-d.MsgNearestPeersAckCh:
			msg := ch.(*Message)
			ps, _ := d.streamPool.GetStream(msg.PeerID)
			v, ok := ps.replys.Load(msg.Code)
			if ok {
				payload := make([]*PeerInfo2, 0)
				_ = rlp.DecodeBytes(msg.Payload, &payload)
				reply := make([]*peerstore.PeerInfo, 0)
				for _, info := range payload {
					reply = append(reply, FromPeerInfo2(info))
				}
				replyCh := v.(chan interface{})
				replyCh <- reply
				log.CLog().WithFields(logrus.Fields{"Size": len(reply)}).Debug("PeerID: ", msg.PeerID)
			}
		}
	}
}

func (d *Discovery) lookup(peerID peer.ID) error {
	var (
		ask         = make([]*peerstore.PeerInfo, ConcurrencyLimit)
		asked       = make(map[peer.ID]bool) // called findnode
		seen        = make(map[peer.ID]bool) // called bond
		reply       = make(chan []*peerstore.PeerInfo, ConcurrencyLimit)
		seenInfos   = make([]*peerstore.PeerInfo, 0)
		bondReply   = make(chan *peerstore.PeerInfo, BucketSize*ConcurrencyLimit)
		askPending  = 0
		bondPending = 0
	)
	asked[d.hostID] = true
	closest := d.routingTable.NearestPeers(kb.ConvertPeerID(peerID), BucketSize)
	closestPeerInfo := make([]*peerstore.PeerInfo, 0, BucketSize)
	for _, id := range closest {
		p := d.peerstore.PeerInfo(id)
		closestPeerInfo = append(closestPeerInfo, &p)
	}
	ask = sortByDistance(closestPeerInfo, peerID)

	for len(ask) > 0 {
		if askPending == 0 {
			for _, v := range ask {
				askPending++
				asked[v.ID] = true
				fmt.Println(v.ID)
				//fmt.Println("asked ", len(asked))
				go d.findnode(v, peerID, reply)
			}
		}
		for _, n := range <-reply {
			fmt.Println("here1")
			fmt.Println(n)
			//if n != nil && !seen[n.ID] && !asked[n.ID] {
			if n != nil && !asked[n.ID] {
				if !seen[n.ID] {
					seen[peerID] = true
					bondPending++
					fmt.Println(bondPending)
					go d.bondReply(n, bondReply)
				}
			} else {
				fmt.Println("here2")
			}
		}
		askPending--
		if askPending == 0 {
			if bondPending == 0 {
				ask = ask[:0]
			} else {
				for n := range bondReply {
					seenInfos = append(seenInfos, n)
					bondPending--
					//fmt.Println(bondPending)
					if bondPending == 0 {
						break
					}
				}
				ask = sortByDistance(seenInfos, peerID)
				seenInfos = seenInfos[:0]
				//fmt.Println("ask ", len(ask))
			}
		}

	}
	close(reply)
	close(bondReply)
	fmt.Println("peerstore size : ", len(d.peerstore.Peers()))
	for _, id := range d.peerstore.Peers() {
		fmt.Println(id.Pretty())
	}
	return nil
}

func (d *Discovery) randomLookup() error {
	//TODO: rlock
	peers := d.routingTable.ListPeers()
	size := len(peers)
	if size == 0 {
		return errors.New("Not found peer")
	}
	rand.Seed(time.Now().Unix())
	id := peers[rand.Intn(size)]

	return d.lookup(id)
}

func (d *Discovery) Start() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.CLog().WithFields(logrus.Fields{
				"count": runtime.NumGoroutine(),
			}).Debug("NumGoroutine")
			err := d.randomLookup()
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Warning("randomLookup")
		}
	}
}

/*
func (nodeRoute *NodeRoute) Start() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.CLog().WithFields(logrus.Fields{
				"count": runtime.NumGoroutine(),
			}).Debug("NumGoroutine")
			nodeRoute.FindNewNodes()
		}
	}
}
*/
