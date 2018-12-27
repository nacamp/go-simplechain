package net

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/najimmy/go-simplechain/log"
	"github.com/sirupsen/logrus"
)

type NodeRoute struct {
	mu           sync.Mutex
	done         chan bool
	node         *Node
	AddrMap      map[peer.ID]ma.Multiaddr // TODO  change []ma.Multiaddr
	routingTable *kb.RoutingTable
}

func NewNodeRoute(node *Node) *NodeRoute {
	nodeRoute := &NodeRoute{node: node, AddrMap: make(map[peer.ID]ma.Multiaddr)}
	nodeRoute.routingTable =
		kb.NewRoutingTable(20, kb.ConvertPeerID(node.host.ID()), time.Minute, node.host.Peerstore())
	return nodeRoute
}

func (nodeRoute *NodeRoute) Update(peerid peer.ID, addr ma.Multiaddr) {
	log.CLog().Debug("Update lock before")
	nodeRoute.mu.Lock()
	nodeRoute.routingTable.Update(peerid)
	nodeRoute.AddrMap[peerid] = addr
	nodeRoute.mu.Unlock()
	log.CLog().Debug("Update unlock after")

	log.CLog().WithFields(logrus.Fields{
		"ID":   peerid,
		"addr": addr,
	}).Debug("")
}

func (nodeRoute *NodeRoute) Remove(peerid peer.ID) {
	log.CLog().Debug("Remove lock before")
	nodeRoute.mu.Lock()
	nodeRoute.routingTable.Remove(peerid)
	delete(nodeRoute.AddrMap, peerid)
	nodeRoute.mu.Unlock()
	log.CLog().Debug("Remove unlock after")
}

func (nodeRoute *NodeRoute) NearestPeers(peerid peer.ID, count int) map[peer.ID]ma.Multiaddr {
	AddrMap := make(map[peer.ID]ma.Multiaddr)
	log.CLog().Debug("NearestPeers  lock before")
	nodeRoute.mu.Lock()
	peers := nodeRoute.routingTable.NearestPeers(kb.ConvertPeerID(peerid), count)
	for i, p := range peers {
		AddrMap[p] = nodeRoute.AddrMap[peers[i]]
	}
	nodeRoute.mu.Unlock()
	log.CLog().Debug("NearestPeers unlock after")
	log.CLog().WithFields(logrus.Fields{
		"peers": peers,
	}).Debug("")
	return AddrMap
}

//AddNodeFromSeedString is
func (nodeRoute *NodeRoute) AddNodeFromSeedString(seed string) {

	ipfsaddr, err := ma.NewMultiaddr(seed)
	if err != nil {
		log.CLog().Warning(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.CLog().Warning(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.CLog().Warning(err)
	}
	log.CLog().WithFields(logrus.Fields{
		"ipfsaddr": ipfsaddr,
		"pid":      pid,
		"peerid":   peerid,
	}).Debug("")

	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr) //  /ip4/127.0.0.1/tcp/9990
	nodeRoute.Update(peerid, targetAddr)
	nodeRoute.node.host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	nodeRoute.node.seedID = peerid
}

func (nodeRoute *NodeRoute) FindNewNodes() {
	log.CLog().Debug("FindNewNodes")
	node := nodeRoute.node
	peers := nodeRoute.NearestPeers(nodeRoute.node.host.ID(), 20)
	for peerid, addr := range peers {
		if peerid == node.host.ID() {
			continue
		}

		v, ok := node.p2pStreamMap.Load(peerid)
		if ok {
			p2pStream := v.(*P2PStream)
			p2pStream.mu.RLock()
			if !p2pStream.isClosed {
				p2pStream.mu.RUnlock()
				log.CLog().Debug("reuse stream")
				p2pStream.RequestPeers()
			} else {
				p2pStream.mu.RUnlock()
				log.CLog().Debug("FindNewNodes lock before")
				p2pStream.mu.Lock()
				node.p2pStreamMap.Delete(peerid)
				node.host.Peerstore().ClearAddrs(peerid)
				nodeRoute.Remove(p2pStream.peerID)
				log.CLog().WithFields(logrus.Fields{
					"ID": p2pStream.peerID,
				}).Debug("P2PStream removed")

				if node.seedID == peerid {
					log.CLog().Debug("add seed node")
					nodeRoute.Update(peerid, addr)
				}
				p2pStream.mu.Unlock()
				log.CLog().Debug("FindNewNodes Unlock after")
			}

		} else {
			// Always firt add id and addr at Peerstore
			node.host.Peerstore().AddAddr(peerid, addr, pstore.PermanentAddrTTL)
			p2pStream, err := NewP2PStream(node, peerid)
			if err != nil {
				node.host.Peerstore().ClearAddrs(peerid)
				nodeRoute.Remove(peerid)
			} else {
				log.CLog().WithFields(logrus.Fields{
					"ID": p2pStream.peerID,
				}).Debug("P2PStream added")
				node.p2pStreamMap.Store(p2pStream.peerID, p2pStream)
				p2pStream.Start(false)
				p2pStream.RequestPeers()
			}

		}
	}
	log.CLog().WithFields(logrus.Fields{
		"size": len(peers),
	}).Debug("Peer size")
}

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
