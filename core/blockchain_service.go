package core

import (
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/sirupsen/logrus"
)

type BlockChainService struct {
	node net.INode
	bc   *BlockChain
}

func NewBlockChainService(bc *BlockChain, node net.INode) *BlockChainService {
	bcs := BlockChainService{
		node: node,
		bc:   bc,
	}
	bcs.registerSubscriber()
	return &bcs
}

func (bcs *BlockChainService) registerSubscriber() {
	node := bcs.node
	node.RegisterSubscriber(net.MSG_NEW_BLOCK, bcs)
	node.RegisterSubscriber(net.MSG_MISSING_BLOCK, bcs)
	node.RegisterSubscriber(net.MSG_MISSING_BLOCK_ACK, bcs)
	node.RegisterSubscriber(net.MSG_NEW_TX, bcs)
}

func (bcs *BlockChainService) Start() {
	go bcs.loop()
	go bcs.messageLoop()
}

func (bcs *BlockChainService) loop() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			bcs.bc.RequestMissingBlock()
		}
	}
}

func (bcs *BlockChainService) messageLoop() {
	for {
		select {
		case msg := <-bcs.bc.MessageToRandomNode:
			bcs.node.SendMessageToRandomNode(msg)
		case msg := <-bcs.bc.NewTXMessage:
			bcs.BroadcastNewTXMessage(msg)
		}
	}
}

func (bcs *BlockChainService) HandleMessage(message *net.Message) error {
	bc := bcs.bc
	if message.Code == net.MSG_NEW_BLOCK || message.Code == net.MSG_MISSING_BLOCK_ACK {
		baseBlock := &BaseBlock{}
		err := rlp.DecodeBytes(message.Payload, baseBlock)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg":  err,
				"Code": message.Code,
			}).Warning("DecodeBytes")
		}
		err = bc.PutBlockIfParentExist(baseBlock.NewBlock())
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg":  err,
				"Code": message.Code,
			}).Warning("PutBlockIfParentExist")
		}
		bc.Consensus.UpdateLIB()
		bc.RemoveOrphanBlock()
	} else if message.Code == net.MSG_MISSING_BLOCK {
		height := uint64(0)
		err := rlp.DecodeBytes(message.Payload, &height)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg":  err,
				"Code": message.Code,
			}).Warning("DecodeBytes")
		}
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Debug("missing block request arrived")
		bcs.SendMissingBlock(height, message.PeerID)
	} else if message.Code == net.MSG_NEW_TX {
		tx := &Transaction{}
		err := rlp.DecodeBytes(message.Payload, &tx)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg":  err,
				"Code": message.Code,
			}).Warning("DecodeBytes")
		}
		log.CLog().WithFields(logrus.Fields{
			"From":   common.Address2Hex(tx.From),
			"To":     common.Address2Hex(tx.To),
			"Amount": tx.Amount,
		}).Info("Received tx")
		bc.TxPool.Put(tx)

	}
	return nil
}

func (bcs *BlockChainService) SendMissingBlock(height uint64, peerID peer.ID) {
	bc := bcs.bc
	block := bc.GetBlockByHeight(height)
	// if err != nil {
	// 	return err
	// }
	if block != nil {
		message, _ := net.NewRLPMessage(net.MSG_MISSING_BLOCK_ACK, block.BaseBlock)
		bcs.node.SendMessage(&message, peerID)
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Info("Send missing block")
	} else {
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Info("We don't have missing block")
	}
}

func (bcs *BlockChainService) BroadcastNewTXMessage(tx *Transaction) error {
	message, err := net.NewRLPMessage(net.MSG_NEW_TX, tx)
	if err != nil {
		return err
	}
	// bc.BroadcastMessage <- &message
	bcs.node.BroadcastMessage(&message)
	return nil
}
