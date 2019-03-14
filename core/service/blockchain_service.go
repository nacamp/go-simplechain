package service

import (
	"fmt"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/sirupsen/logrus"
)

type BlockChainService struct {
	// node                 net.INode
	bc                    *core.BlockChain
	streamPool            *net.PeerStreamPool
	MsgNewBlockCh         chan interface{}
	MsgMissingBlockCh     chan interface{}
	MsgMissingBlockAckCh  chan interface{}
	MsgMissingBlocksCh    chan interface{}
	MsgMissingBlocksAckCh chan interface{}
	MsgNewTxCh            chan interface{}
}

func NewBlockChainService(bc *core.BlockChain, streamPool *net.PeerStreamPool) *BlockChainService {
	bcs := BlockChainService{
		// node: node,
		streamPool: streamPool,
		bc:         bc,
	}
	bcs.MsgNewBlockCh = make(chan interface{}, 1)
	bcs.MsgMissingBlockCh = make(chan interface{}, 1)
	bcs.MsgMissingBlockAckCh = make(chan interface{}, 1)
	bcs.MsgMissingBlocksCh = make(chan interface{}, 1)
	bcs.MsgMissingBlocksAckCh = make(chan interface{}, 1)
	bcs.MsgNewTxCh = make(chan interface{}, 1)
	return &bcs
}

func (bcs *BlockChainService) Register(peerStream *net.PeerStream) {
	peerStream.Register(net.MsgNewBlock, bcs.MsgNewBlockCh)
	peerStream.Register(net.MsgMissingBlock, bcs.MsgMissingBlockCh)
	peerStream.Register(net.MsgMissingBlockAck, bcs.MsgMissingBlockAckCh)
	peerStream.Register(net.MsgMissingBlocks, bcs.MsgMissingBlocksCh)
	peerStream.Register(net.MsgMissingBlocksAck, bcs.MsgMissingBlocksAckCh)
	peerStream.Register(net.MsgNewTx, bcs.MsgNewTxCh)
}

func (bcs *BlockChainService) StartHandler() {
	go bcs.onHandle()
}

func (bcs *BlockChainService) Start() {
	go bcs.loop()
}

func (bcs *BlockChainService) loop() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			bcs.bc.RequestMissingBlocks()
		case msg := <-bcs.bc.MessageToRandomNode:
			bcs.streamPool.SendMessageToRandomNode(msg)
		case msg := <-bcs.bc.BroadcastMessage:
			bcs.streamPool.BroadcastMessage(msg)
		case msg := <-bcs.bc.NewTXMessage:
			bcs.BroadcastNewTXMessage(msg)
		}
	}
}

func (bcs *BlockChainService) receiveBlock(msg *net.Message, isNew bool) {
	bc := bcs.bc
	// msg := ch.(*Message)
	baseBlock := &core.BaseBlock{}
	err := rlp.DecodeBytes(msg.Payload, baseBlock)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{"Code": msg.Code}).Warning(fmt.Sprintf("%+v", err))
	}
	// rlp.DecodeBytes(message.Payload, &data)
	log.CLog().WithFields(logrus.Fields{}).Debug("PeerID: ", msg.PeerID)

	err = bc.PutBlockIfParentExist(baseBlock.NewBlock())
	if err != nil {
		log.CLog().WithFields(logrus.Fields{"Code": msg.Code}).Warning(fmt.Sprintf("%+v", err))
	}
	bc.Consensus.UpdateLIB()
	bc.RemoveOrphanBlock()
	if isNew {
		bcs.streamPool.BroadcastMessage(msg)
	}
}

func (bcs *BlockChainService) onHandle() {
	bc := bcs.bc
	for {
		select {
		case ch := <-bcs.MsgMissingBlockAckCh:
			bcs.receiveBlock(ch.(*net.Message), false)
		case ch := <-bcs.MsgMissingBlocksAckCh:
			bcs.receiveBlock(ch.(*net.Message), false)
		case ch := <-bcs.MsgNewBlockCh:
			bcs.receiveBlock(ch.(*net.Message), true)
		case ch := <-bcs.MsgMissingBlockCh:
			msg := ch.(*net.Message)
			hash := common.Hash{}
			err := rlp.DecodeBytes(msg.Payload, &hash)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{"Code": msg.Code}).Warning(fmt.Sprintf("%+v", err))
			}
			log.CLog().WithFields(logrus.Fields{
				"Hash": common.HashToHex(hash),
			}).Debug("missing block request arrived")
			bcs.SendMissingBlock(hash, msg.PeerID)

		case ch := <-bcs.MsgMissingBlocksCh:
			msg := ch.(*net.Message)
			var blockRange [2]uint64
			err := rlp.DecodeBytes(msg.Payload, &blockRange)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{"Code": msg.Code}).Warning(fmt.Sprintf("%+v", err))
			}
			log.CLog().WithFields(logrus.Fields{
				"Height Start": blockRange[0],
				"Height End":   blockRange[1],
			}).Debug("missing block request arrived")
			bcs.SendMissingBlocks(blockRange, msg.PeerID)

		case ch := <-bcs.MsgNewTxCh:
			msg := ch.(*net.Message)
			tx := &core.Transaction{}
			err := rlp.DecodeBytes(msg.Payload, &tx)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{"Code": msg.Code}).Warning(fmt.Sprintf("%+v", err))
			}
			log.CLog().WithFields(logrus.Fields{
				"From":   common.AddressToHex(tx.From),
				"To":     common.AddressToHex(tx.To),
				"Amount": tx.Amount,
			}).Info("Received tx")
			bc.TxPool.Put(tx)
			bcs.streamPool.BroadcastMessage(msg)
		}
	}
}

func (bcs *BlockChainService) SendMissingBlock(hash common.Hash, peerID peer.ID) {
	bc := bcs.bc
	block := bc.GetBlockByHash(hash)
	if block != nil {
		message, _ := net.NewRLPMessage(net.MsgMissingBlockAck, block.BaseBlock)
		ps, err := bcs.streamPool.GetStream(peerID)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
		}
		ps.SendMessage(&message)
		log.CLog().WithFields(logrus.Fields{
			"Height": block.Header.Height,
			"Hash":   common.HashToHex(hash),
		}).Info("Send missing block")
	} else {
		log.CLog().WithFields(logrus.Fields{
			"Hash": common.HashToHex(hash),
		}).Info("We don't have missing block")
	}
}

func (bcs *BlockChainService) SendMissingBlocks(blockRange [2]uint64, peerID peer.ID) {
	bc := bcs.bc
	ps, err := bcs.streamPool.GetStream(peerID)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
	}
	for i := blockRange[0]; i <= blockRange[1]; i++ {
		block := bc.GetBlockByHeight(i)
		if block != nil {
			message, _ := net.NewRLPMessage(net.MsgMissingBlockAck, block.BaseBlock)
			ps.SendMessage(&message)
			log.CLog().WithFields(logrus.Fields{
				"Height": block.Header.Height,
				"Hash":   common.HashToHex(block.Hash()),
			}).Info("Send missing block")
		} else {
			log.CLog().WithFields(logrus.Fields{
				"Hash": common.HashToHex(block.Hash()),
			}).Info("We don't have missing block")
		}
	}

}

func (bcs *BlockChainService) BroadcastNewTXMessage(tx *core.Transaction) error {
	message, err := net.NewRLPMessage(net.MsgNewTx, tx)
	if err != nil {
		return err
	}
	bcs.streamPool.BroadcastMessage(&message)
	return nil
}
