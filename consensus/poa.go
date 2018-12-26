package consensus

import (
	"crypto/ecdsa"
	"errors"
	"sync"
	"time"

	"github.com/najimmy/go-simplechain/rlp"

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
	mu           sync.RWMutex
	bc           *core.BlockChain
	node         *net.Node
	coinbase     common.Address
	priv         *ecdsa.PrivateKey
	enableMining bool
	Storage      storage.Storage
	voteTx       *core.Transaction
	Period       uint64
}

func NewPoa(storage storage.Storage) *Poa {
	return &Poa{Storage: storage}
}

//Same as dpos
func (cs *Poa) Setup(bc *core.BlockChain, node *net.Node, address common.Address, bpriv []byte, period int) {
	cs.bc = bc
	cs.node = node
	cs.enableMining = true
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), bpriv)
	cs.coinbase = common.BytesToAddress(pub.SerializeCompressed())
	cs.priv = (*ecdsa.PrivateKey)(priv)
	cs.Period = uint64(period)
	if cs.coinbase != address {
		log.CLog().WithFields(logrus.Fields{
			"Address": common.Address2Hex(cs.coinbase),
		}).Panic("Privatekey is different")
	}
}

//Same as dpos
func (dpos *Poa) SetupNonMiner(bc *core.BlockChain, node *net.Node) {
	dpos.bc = bc
	dpos.node = node
}

//To be changed
func (cs *Poa) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	//Fix: when ticker is 1 second, server mining...
	// turn := (now % 9) / 3
	block, err := bc.NewBlockFromParent(bc.Tail)
	if err != nil {
		log.CLog().Warning(err)
	}
	block.Header.Time = now
	miners, err := cs.GetMiners(bc.Tail.Hash())
	if len(miners) == 0 {
		log.CLog().WithFields(logrus.Fields{
			"Size": 0,
		}).Panic("Miner must be one more")
	}
	turn := (now % (uint64(len(miners)) * cs.Period)) / cs.Period
	snapshot, err := cs.snapshot(block.Header.ParentHash)
	// minerGroup, _, err := block.MinerState.GetMinerGroup(bc, block)
	if err != nil {
		log.CLog().Warning(err)
	}
	if snapshot == nil {
		log.CLog().WithFields(logrus.Fields{
			"Height":     block.Header.Height,
			"ParentHash": common.Hash2Hex(block.Header.ParentHash),
		}).Warning("Snapshot is nil")
		return nil
	}
	if snapshot.SignerSlice()[turn] == cs.coinbase {
		parent := bc.GetBlockByHash(bc.Tail.Header.ParentHash)

		if (parent != nil) && (now-parent.Header.Time < cs.Period) { //(3 * 3)
			log.CLog().WithFields(logrus.Fields{
				"address": common.Address2Hex(cs.coinbase),
			}).Warning("Interval is short")
			return nil
		}

		log.CLog().WithFields(logrus.Fields{
			"address": common.Bytes2Hex(cs.coinbase[:]),
		}).Debug("my turn")
		block.Header.Coinbase = cs.coinbase
		//block.Header.SnapshotVoterTime = bc.Tail.Header.SnapshotVoterTime // voterBlock.Header.Time
		//because PutMinerState recall GetMinerGroup , here assign  bc.Tail.Header.SnapshotVoterTime , not voterBlock.Header.Time

		//TODO: check double spending ?
		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		voteCount := 0
		for i := 0; i < bc.TxPool.Len(); i++ {
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
				// if signer is miner, include  voting tx
				if len(tx.Payload) > 0 {
					if tx.From == cs.coinbase {
						if voteCount == 0 {
							voteCount++
							block.Transactions = append(block.Transactions, tx)
						} else {
							bc.TxPool.Put(tx)
						}
					} else {
						bc.TxPool.Put(tx)
					}
				} else {
					block.Transactions = append(block.Transactions, tx)
				}
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
		//TODO: snapshot hash
		block.MakeHash()
		newSnap := snapshot.Copy()
		newSnap.BlockHash = block.Hash()
		l := len(newSnap.Signers)
		if voteCount > 0 && cs.voteTx != nil {
			authorize := bool(true)
			rlp.DecodeBytes(cs.voteTx.Payload, &authorize)
			if newSnap.Cast(cs.coinbase, cs.voteTx.To, authorize) {
				newSnap.Apply()
			}
		}
		if l != len(newSnap.Signers) {
			log.CLog().WithFields(logrus.Fields{
				"Size": len(newSnap.Signers),
			}).Info("changed signers")
		}
		newSnap.Store(cs.Storage)
		cs.voteTx = nil
		return block
	} else {
		log.CLog().WithFields(logrus.Fields{
			"address": common.Bytes2Hex(cs.coinbase[:]),
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
	ticker := time.NewTicker(2 * time.Second)
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
	return LoadSnapshot(poa.Storage, hash)
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
func (c *Poa) ConsensusType() string {
	return "POA"
}

func (c *Poa) ExecuteVote(tx *core.Transaction) {
	c.voteTx = tx
}

//TOD change name
func (c *Poa) InitSaveSnapshot(hash common.Hash, addresses []common.Address) {
	snap := NewSnapshot(hash, addresses)
	snap.Store(c.Storage)
}

func (cs *Poa) GetMiners(hash common.Hash) ([]common.Address, error) {
	snap, err := cs.snapshot(hash)
	if err != nil {
		return nil, err
	}
	return snap.SignerSlice(), nil
}

func (cs *Poa) SaveMiners(block *core.Block) error {
	snapshot, err := cs.snapshot(block.Header.ParentHash)
	// minerGroup, _, err := block.MinerState.GetMinerGroup(bc, block)
	if err != nil {
		log.CLog().Warning(err)
		return err
	}
	if snapshot == nil {
		return errors.New("Snapshot is nil")
	}
	newSnap := snapshot.Copy()
	newSnap.BlockHash = block.Hash()
	if cs.voteTx != nil {
		authorize := bool(true)
		rlp.DecodeBytes(cs.voteTx.Payload, &authorize)
		if newSnap.Cast(cs.voteTx.From, cs.voteTx.To, authorize) {
			newSnap.Apply()
		}
	}
	newSnap.Store(cs.Storage)
	cs.voteTx = nil
	return nil
}

func (cs *Poa) VerifyMinerTurn(block *core.Block) error {
	parentBlock := cs.bc.GetBlockByHash(block.Header.ParentHash)
	if parentBlock == nil {
		return errors.New("parent block is nil")
	}
	miners, err := cs.GetMiners(parentBlock.Hash())
	if err != nil {
		return err
	}
	index := (block.Header.Time % (uint64(len(miners)) * cs.Period)) / cs.Period
	if miners[index] != block.Header.Coinbase {
		return errors.New("This turn is not this miner's turn ")
	}
	return nil
}
