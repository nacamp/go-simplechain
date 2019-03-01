package poa

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/tests"
)

func TestPoa(t *testing.T) {
	var err error
	//config
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	mstrg, _ := storage.NewMemoryStorage()

	cs := NewPoa(net.NewPeerStreamPool(), mstrg)
	bc := core.NewBlockChain(mstrg)

	//test MakeGenesisBlock in Setup
	bc.Setup(cs, voters)
	state, _ := NewInitState(cs.bc.GenesisBlock.ConsensusState.RootHash(), 0, mstrg)
	_, err = state.Signer.Get(common.FromHex(tests.Addr0))
	assert.NoError(t, err)
	_, err = state.Signer.Get(common.FromHex(tests.Addr1))
	assert.NoError(t, err)
	_, err = state.Signer.Get(common.FromHex(tests.Addr2))
	assert.NoError(t, err)

}

/*

//To be changed
func (cs *Poa) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	block, err := bc.NewBlockFromTail()
	if err != nil {
		log.CLog().Warning(err)
	}

	block.Header.Time = now
	state := block.ConsensusState.(*PoaState)
	// miners, err := cs.GetMiners(block.Header.ParentHash)
	miners, err := state.GetMiners()
	if len(miners) == 0 {
		log.CLog().WithFields(logrus.Fields{
			"Size": 0,
		}).Panic("Miner must be one more")
	}
	if err != nil {
		log.CLog().Warning(err)
	}
	turn := (now % (uint64(len(miners)) * cs.Period)) / cs.Period
	// snapshot := block.Snapshot.(*PoaState)
	// // snapshot, err := cs.Snapshot(block.Header.ParentHash)
	// if err != nil {
	// 	log.CLog().Warning(err)
	// }
	// if snapshot == nil {
	// 	log.CLog().WithFields(logrus.Fields{
	// 		"Height":     block.Header.Height,
	// 		"ParentHash": common.HashToHex(block.Header.ParentHash),
	// 	}).Warning("Snapshot is nil")
	// 	return nil
	// }
	// if snapshot.SignerSlice()[turn] == cs.coinbase {
	if miners[turn] == cs.coinbase {
		parent := bc.GetBlockByHash(block.Header.ParentHash)

		//if (parent != nil) && (now-parent.Header.Time < ((uint64(len(miners)) * cs.Period) - 1)) { //(3 * 3)
		if (parent != nil) && (now-parent.Header.Time < cs.Period) { //(3 * 3)
			log.CLog().WithFields(logrus.Fields{
				"address": common.AddressToHex(cs.coinbase),
			}).Debug("Interval is short")
			return nil
		}

		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("my turn")
		block.Header.Coinbase = cs.coinbase

		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		firstVote := true
		// var voteTx *core.Transaction
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
					"Address": common.AddressToHex(tx.From),
				}).Warning("Not found account")
			} else if fromAccount.Nonce+1 == tx.Nonce {
				// if signer is miner, include  voting tx
				if tx.Payload != nil {
					if tx.From == cs.coinbase && firstVote {
						firstVote = false
						// voteTx = tx
						block.Transactions = append(block.Transactions, tx)
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
					"Address": common.AddressToHex(tx.From),
				}).Warning("cannot accept a transaction with wrong nonce")
			}
		}
		bc.RewardForCoinbase(block)
		bc.ExecuteTransaction(block)
		block.Header.AccountHash = block.AccountState.RootHash()
		block.Header.TransactionHash = block.TransactionState.RootHash()
		cs.SaveState(block)
		if err := cs.Verify(block); err != nil {
			log.CLog().WithFields(logrus.Fields{
				"address": common.BytesToHex(cs.coinbase[:]),
			}).Debug("not my turn")
		}
		block.Header.ConsensusHash = state.RootHash()
		// _ = voteTx
		// newSnap := snapshot.Copy()
		// l := len(newSnap.Signers)
		// if voteTx != nil {
		// 	//TODO: fix after dpos coding
		// 	// authorize := bool(true)
		// 	// rlp.DecodeBytes(voteTx.Payload, &authorize)
		// 	// if newSnap.Cast(cs.coinbase, voteTx.To, authorize) {
		// 	// 	newSnap.Apply()
		// 	// }
		// }
		// if l != len(newSnap.Signers) {
		// 	log.CLog().WithFields(logrus.Fields{
		// 		"Size": len(newSnap.Signers),
		// 	}).Info("changed signers")
		// }
		// block.Header.SnapshotHash = newSnap.CalcHash()
		block.MakeHash()
		// newSnap.BlockHash = block.Hash()
		// newSnap.Store(cs.Storage)
		// //this code need, because of rlp encoding
		// block.Snapshot = nil
		return block
	} else {
		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("not my turn")
		return nil
	}
}

func (cs *Poa) Start() {
	if cs.enableMining {
		go cs.loop()
	}
}

func (cs *Poa) loop() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			block := cs.MakeBlock(uint64(now.Unix()))
			if block != nil {
				sig, err := cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
				if err != nil {
					log.CLog().WithFields(logrus.Fields{
						"Msg": err,
					}).Warning("SignHash")
				}
				block.SignWithSignature(sig)
				cs.bc.PutBlockByCoinbase(block)
				cs.bc.Consensus.UpdateLIB()
				cs.bc.RemoveOrphanBlock()
				message, _ := net.NewRLPMessage(net.MsgNewBlock, block.BaseBlock)
				cs.streamPool.BroadcastMessage(&message)
			}
		}
	}
}


func (cs *Poa) getMinerSize(block *core.Block) (minerSize int, err error) {
	parentBlock := cs.bc.GetBlockByHash(block.Header.ParentHash)
	if parentBlock == nil {
		return 0, errors.New("Parent is nil")
	}
	block.ConsensusState, err = cs.LoadState(parentBlock)
	if err != nil {
		return 0, err
	}
	state := block.ConsensusState.(*PoaState)
	ms, err := state.GetMiners()
	if err != nil {
		return 0, err
	}
	minerSize = len(ms)
	if minerSize == 0 {
		return 0, errors.New("Miners length cannot is zero")
	}
	return minerSize, nil
}



//----------    Consensus  ----------------//
func (cs *Poa) UpdateLIB() {
	bc := cs.bc
	block := bc.Tail
	//FIXME: consider timestamp
	miners := make(map[common.Address]bool)
	turn := 1
	if block.Header.Height == 0 {
		return
	}
	firstMinerSize, err := cs.getMinerSize(block)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{
			"Msg": err,
		}).Warning("getMinerSize")
		return
	}
	if firstMinerSize < 3 {
		log.CLog().WithFields(logrus.Fields{
			"Size": firstMinerSize,
		}).Debug("At least 3 node are needed")
		return
	}
	for bc.Lib.Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		size, err := cs.getMinerSize(block)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Warning("getMinerSize")
			return
		}
		if firstMinerSize != size {
			return
		}
		if turn == firstMinerSize {
			if len(miners) == firstMinerSize*2/3+1 {
				bc.SetLib(block)
				log.CLog().WithFields(logrus.Fields{
					"Height": block.Header.Height,
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

func (cs *Poa) ConsensusType() string {
	return "POA"
}




func (cs *Poa) Verify(block *core.Block) error {
	// parentBlock := cs.bc.GetBlockByHash(block.Header.ParentHash)
	// if parentBlock == nil {
	// 	return errors.New("parent block is nil")
	// }
	// miners, err := cs.GetMiners(parentBlock.Hash())
	// if err != nil {
	// 	return err
	// }

	state := block.ConsensusState.(*PoaState)
	miners, err := state.GetMiners()
	if err != nil {
		return err
	}
	index := (block.Header.Time % (uint64(len(miners)) * cs.Period)) / cs.Period
	if miners[index] != block.Header.Coinbase {
		return errors.New("This turn is not this miner's turn ")
	}
	return nil
}



func (cs *Poa) SaveState(block *core.Block) (err error) {
	state := block.ConsensusState.(*PoaState)
	state.RefreshSigner()
	// accs := block.AccountState

	// electedTime := state.GetNewElectedTime(state.ElectedTime, block.Header.Time, 3, 3, 3)

	// if electedTime == block.Header.Time {
	// 	miners, err := state.GetNewRoundMiners(block.Header.Time, 3)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	state.MinersHash, err = state.PutMiners(miners)
	// 	state.ElectedTime = block.Header.Time

	// 	iter, err := state.Voter.Iterator(nil)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	exist, _ := iter.Next()
	// 	for exist {
	// 		account := accs.GetAccount(common.BytesToAddress(iter.Key()))
	// 		account.CalcSetTotalPeggedStake()
	// 		accs.PutAccount(account)
	// 		exist, err = iter.Next()
	// 	}
	// 	//reset voter if this round is new
	// 	state.Voter, err = trie.NewTrie(nil, cs.bc.Storage, false)
	// }
	err = state.Put(block.Header.Height)
	if err != nil {
		return err
	}
	return nil

	// if err := cs.VerifyMinerTurn(block); err != nil {
	// 	return err
	// }
	// snapshot, err := cs.LoadSnapshot(block.Header.ParentHash)
	// if err != nil {
	// 	log.CLog().Warning(err)
	// 	return err
	// }
	// if snapshot == nil {
	// 	return errors.New("Snapshot is nil")
	// }
	// newSnap := snapshot.Copy()
	// newSnap.BlockHash = block.Hash()
	// for _, tx := range block.Transactions {

	// 	if tx.Payload != nil {
	// 		//TODO: fix after dpos coding
	// 		// authorize := bool(true)
	// 		// rlp.DecodeBytes(tx.Payload, &authorize)
	// 		// if newSnap.Cast(tx.From, tx.To, authorize) {
	// 		// 	newSnap.Apply()
	// 		// }
	// 		break
	// 	}
	// }
	// h := newSnap.CalcHash()
	// if h != block.Header.SnapshotHash {
	// 	return errors.New("Hash is different")
	// }
	// newSnap.Store(cs.Storage)
	return nil
}



func (cs *Poa) LoadState(block *core.Block) (state core.ConsensusState, err error) {
	bc := cs.bc

	state, err = NewInitState(block.Header.ConsensusHash, block.Header.Height, bc.Storage)
	if err != nil {
		return nil, err
	}
	return state, nil
}

*/