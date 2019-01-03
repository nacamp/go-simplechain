package core

import (
	"time"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/rlp"
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
	bc := bcs.bc
	node.RegisterSubscriber(net.MSG_NEW_BLOCK, bc)
	node.RegisterSubscriber(net.MSG_MISSING_BLOCK, bc)
	node.RegisterSubscriber(net.MSG_MISSING_BLOCK_ACK, bc)
	node.RegisterSubscriber(net.MSG_NEW_TX, bc)
}

func (bcs *BlockChainService) Start() {
	go bcs.loop()
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
		bc.SendMissingBlock(height, message.PeerID)
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
