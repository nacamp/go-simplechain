package consensus

import (
	"crypto/ecdsa"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/trie"
	"github.com/sirupsen/logrus"
)

type Poa struct {
	bc           *core.BlockChain
	node         *net.Node
	coinbase     common.Address
	priv         *ecdsa.PrivateKey
	enableMining bool
}

func NewPos() *Poa {
	return &Poa{}
}

//Same as dpos
func (poa *Poa) Setup(bc *core.BlockChain, node *net.Node, address common.Address, bpriv []byte) {
	poa.bc = bc
	poa.node = node
	poa.enableMining = true
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), bpriv)
	poa.coinbase = common.BytesToAddress(pub.SerializeCompressed())
	poa.priv = (*ecdsa.PrivateKey)(priv)
	if poa.coinbase != address {
		log.CLog().WithFields(logrus.Fields{
			"Address": common.Address2Hex(poa.coinbase),
		}).Panic("Privatekey is different")
	}
}

//Same as dpos
func (dpos *Poa) SetupNonMiner(bc *core.BlockChain, node *net.Node) {
	dpos.bc = bc
	dpos.node = node
}

//To be changed
func (dpos *Poa) MakeBlock(now uint64) *core.Block {
	bc := dpos.bc
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	//Fix: when ticker is 1 second, server mining...
	turn := (now % 9) / 3
	block, err := bc.NewBlockFromParent(bc.Tail)
	if err != nil {
		log.CLog().Warning(err)
	}
	block.Header.Time = now

	//이곳에 마이너 그룹을 가져온다.
	snapshot, err := dpos.snapshot(block.Header.ParentHash)
	// minerGroup, _, err := block.MinerState.GetMinerGroup(bc, block)
	if err != nil {
		log.CLog().Warning(err)
	}

	if snapshot.SignerSlice()[turn] == dpos.coinbase {
		parent := bc.GetBlockByHash(bc.Tail.Header.ParentHash)

		if (parent != nil) && (now-parent.Header.Time < 3) { //(3 * 3)
			log.CLog().WithFields(logrus.Fields{
				"address": common.Address2Hex(dpos.coinbase),
			}).Warning("Interval is short")
			return nil
		}

		log.CLog().WithFields(logrus.Fields{
			"address": common.Bytes2Hex(dpos.coinbase[:]),
		}).Debug("my turn")
		block.Header.Coinbase = dpos.coinbase
		//block.Header.SnapshotVoterTime = bc.Tail.Header.SnapshotVoterTime // voterBlock.Header.Time
		//because PutMinerState recall GetMinerGroup , here assign  bc.Tail.Header.SnapshotVoterTime , not voterBlock.Header.Time

		//TODO: check double spending ?
		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		for {
			tx := bc.TxPool.Pop()
			if tx == nil {
				break
			}
			//TODO: remove code duplicattion in ExecuteTransaction
			fromAccount := accs.GetAccount(tx.From)
			//TODO: check at txpool
			if fromAccount == nil {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.Address2Hex(tx.From),
				}).Warning("Not found account")
			} else if fromAccount.Nonce+1 == tx.Nonce {
				block.Transactions = append(block.Transactions, tx)
			} else if fromAccount.Nonce+1 < tx.Nonce {
				//use in future
				bc.TxPool.Put(tx)
			} else {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.Address2Hex(tx.From),
				}).Warning("cannot accept a transaction with wrong nonce")
			}
		}
		bc.RewardForCoinbase(block)
		bc.ExecuteTransaction(block)
		block.Header.AccountHash = block.AccountState.RootHash()
		block.Header.TransactionHash = block.TransactionState.RootHash()
		// need voterHash at PutMinerState(GetMinerGroup)
		// block.Header.VoterHash = block.VoterState.RootHash()
		// bc.PutMinerState(block)
		// block.Header.MinerHash = block.MinerState.RootHash()
		newSnap := snapshot.Copy()
		if newSnap.Cast(dpos.coinbase, dpos.coinbase, true) {
			newSnap.Apply()
		}
		newSnap.Store(bc.Storage)
		block.MakeHash()
		return block
	} else {
		log.CLog().WithFields(logrus.Fields{
			"address": common.Bytes2Hex(dpos.coinbase[:]),
		}).Debug("not my turn")
		return nil
	}
}

func (dpos *Poa) Start() {
	if dpos.enableMining {
		go dpos.loop()
	}
}

func (dpos *Poa) loop() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			block := dpos.MakeBlock(uint64(now.Unix()))
			if block != nil {
				block.Sign(dpos.priv)
				dpos.bc.PutBlockByCoinbase(block)
				dpos.bc.Consensus.UpdateLIB(dpos.bc)
				dpos.bc.RemoveOrphanBlock()
				message, _ := net.NewRLPMessage(net.MSG_NEW_BLOCK, block.BaseBlock)
				dpos.node.BroadcastMessage(&message)
			}
		}
	}
}

func (poa *Poa) snapshot(hash common.Hash) (*Snapshot, error) {
	block := poa.bc.GetBlockByHash(hash)
	if block.Header.Height == uint64(0) {
		return NewSnapshot(hash, poa.bc.Signers), nil
	}
	return LoadSnapshot(poa.bc.Storage, hash)
}

//---------- Consensus
func (d *Poa) NewMinerState(rootHash common.Hash, storage storage.Storage) (core.MinerState, error) {
	tr, err := trie.NewTrie(common.HashToBytes(rootHash), storage, false)
	return &MinerState{
		Trie: tr,
	}, err
}

func (d *Poa) UpdateLIB(bc *core.BlockChain) {
	block := bc.Tail
	//FIXME: consider timestamp, changed minerGroup
	miners := make(map[common.Address]bool)
	turn := 1
	for bc.Lib.Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		//minerGroup, _, _ := block.MinerState.GetMinerGroup(bc, block)
		if turn == 3 {
			if len(miners) == 3 {
				bc.SetLib(block)
				log.CLog().WithFields(logrus.Fields{
					"Height": block.Header.Height,
					//"address": common.Hash2Hex(block.Hash()),
				}).Info("Updated Lib")
				return
			}
			miners = make(map[common.Address]bool)
			miners[block.Header.Coinbase] = true
			turn = 0
		}
		block = bc.GetBlockByHash(block.Header.ParentHash)
		turn++
	}
	return
}
func (d *Poa) ConsensusType() string {
	return "POA"
}
