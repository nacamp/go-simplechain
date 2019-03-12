//use nebulasio code temporarily
//https://github.com/nebulasio/go-nebulas/blob/feature/nbredev/net/recved_message.go
package net

import (
	"fmt"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/willf/bloom"
)

const (
	// according to https://krisives.github.io/bloom-calculator/
	// Count (n) = 100000, Error (p) = 0.001
	maxCountOfRecvMessageInBloomFiler = 1000000
	bloomFilterOfRecvMessageArgM      = 14377588
	bloomFilterOfRecvMessageArgK      = 10
)

//singletone ?
var (
	bloomFilterOfRecvMessage        = bloom.New(bloomFilterOfRecvMessageArgM, bloomFilterOfRecvMessageArgK)
	bloomMu                         sync.Mutex
	countOfRecvMessageInBloomFilter = 0
)

// RecordKey add key to bloom filter.
func RecordKey(key string) {
	bloomMu.Lock()
	defer bloomMu.Unlock()

	countOfRecvMessageInBloomFilter++
	if countOfRecvMessageInBloomFilter > maxCountOfRecvMessageInBloomFiler {
		// reset.
		// logging.VLog().WithFields(logrus.Fields{
		// 	"countOfRecvMessageInBloomFilter": countOfRecvMessageInBloomFilter,
		// }).Debug("reset bloom filter.")
		countOfRecvMessageInBloomFilter = 0
		bloomFilterOfRecvMessage = bloom.New(bloomFilterOfRecvMessageArgM, bloomFilterOfRecvMessageArgK)
	}

	bloomFilterOfRecvMessage.AddString(key)
}

// HasKey use bloom filter to check if the key exists quickly
func HasKey(key string) bool {
	bloomMu.Lock()
	defer bloomMu.Unlock()

	return bloomFilterOfRecvMessage.TestString(key)
}

// RecordRecvMessage records received message
func RecordRecvMessage(peerID peer.ID, hash uint32) {
	RecordKey(fmt.Sprintf("%s-%d", peerID, hash))
}

// HasRecvMessage check if the received message exists before
func HasRecvMessage(peerID peer.ID, hash uint32) bool {
	return HasKey(fmt.Sprintf("%s-%d", peerID, hash))
}
