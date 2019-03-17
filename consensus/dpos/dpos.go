package dpos

import (
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/trie"

	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/sirupsen/logrus"
)

type Dpos struct {
	bc           *core.BlockChain
	coinbase     common.Address
	wallet       *account.Wallet
	enableMining bool
	streamPool   *net.PeerStreamPool

	period      uint64
	round       uint64
	totalMiners uint64
}

func NewDpos(streamPool *net.PeerStreamPool, period, round, totalMiners uint64) *Dpos {
	return &Dpos{streamPool: streamPool,
		period:      period,
		round:       round,
		totalMiners: totalMiners}
}

func (cs *Dpos) SetupMining(address common.Address, wallet *account.Wallet) {
	cs.enableMining = true
	cs.coinbase = address
	cs.wallet = wallet
}

func (cs *Dpos) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	//Fix: when ticker is 1 second, server mining...
	turn := (now % (cs.totalMiners * cs.period)) / cs.period
	block, err := bc.NewBlockFromTail()
	if err != nil {
		log.CLog().Warning(fmt.Sprintf("%+v", err))
	}
	block.Header.Time = now
	state := block.ConsensusState().(*DposState)
	electedTime := GetNewElectedTime(state.ElectedTime, block.Header.Time, cs.period, cs.round, cs.totalMiners)
	var minerGroup []common.Address
	if electedTime != now {
		minerGroup, err = state.GetMiners(state.MinersHash)
		//minerGroup, _, err := block.MinerState.GetMinerGroup(bc, block)
		if err != nil {
			log.CLog().Warning(fmt.Sprintf("%+v", err))
		}
	}
	if electedTime == now || minerGroup[turn] == cs.coinbase {
		//parent := bc.GetBlockByHash(bc.Tail().Header.ParentHash)
		parent := bc.Tail()
		if (parent != nil) && (now-parent.Header.Time < cs.period) { //(3 * 3)
			log.CLog().WithFields(logrus.Fields{
				"address": common.AddressToHex(cs.coinbase),
			}).Debug("Interval is short")
			return nil
		}

		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("my turn")
		block.Header.Coinbase = cs.coinbase

		//TODO: check double spending ?
		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		noncePool := make(map[common.Address][]*core.Transaction)
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
				block.Transactions = append(block.Transactions, tx)
			} else if fromAccount.Nonce+1 < tx.Nonce {
				v, ok := noncePool[tx.From]
				if ok == true {
					noncePool[tx.From] = append(v, tx)
				} else {
					noncePool[tx.From] = []*core.Transaction{tx}
				}
			} else {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.AddressToHex(tx.From),
				}).Warning("cannot accept a transaction with wrong nonce")
			}
		}
		for k, v := range noncePool {
			sort.Slice(v, func(i, j int) bool {
				return v[i].Nonce < v[j].Nonce
			})
			fromAccount := accs.GetAccount(k)
			nonce := fromAccount.Nonce + 2
			for _, tx := range v {
				if nonce == tx.Nonce {
					block.Transactions = append(block.Transactions, tx)
					nonce++
				} else {
					//use in future
					bc.TxPool.Put(tx)
				}
			}
		}

		for _, tx := range block.Transactions {
			tx.Height = block.Header.Height
		}

		bc.RewardForCoinbase(block)
		bc.ExecuteTransaction(block)
		cs.SaveState(block)
		if err := cs.Verify(block); err != nil {
			log.CLog().WithFields(logrus.Fields{
				"address": common.BytesToHex(cs.coinbase[:]),
			}).Debug("not my turn")
			return nil
		}
		//we need to create an AccountHash after SaveState because the AccountState may change in SaveState.
		block.Header.AccountHash = block.AccountState.RootHash()
		block.Header.TransactionHash = block.TransactionState.RootHash()
		block.Header.ConsensusHash = state.RootHash()
		block.MakeHash()
		return block
	} else {
		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("not my turn")
		return nil
	}
}

func (cs *Dpos) loop() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			block := cs.MakeBlock(uint64(now.Unix()))
			if block != nil {
				sig, err := cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
				if err != nil {
					log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
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

//----------    Consensus  ----------------//

func (dpos *Dpos) Start() {
	if dpos.enableMining {
		go dpos.loop()
	}
}

func (cs *Dpos) Verify(block *core.Block) (err error) {
	//block.Header.Coinbase
	state := block.ConsensusState().(*DposState)
	miners, err := state.GetMiners(state.MinersHash)
	if err != nil {
		return err
	}
	turn := (block.Header.Time % (cs.totalMiners * cs.period)) / cs.period
	if miners[turn] != block.Header.Coinbase {
		return errors.New("This time is not your turn")
	}
	return nil
}

// not use this at GenesisBlock
func (cs *Dpos) SaveState(block *core.Block) (err error) {
	state := block.ConsensusState().(*DposState)
	accs := block.AccountState
	electedTime := GetNewElectedTime(state.ElectedTime, block.Header.Time, cs.period, cs.round, cs.totalMiners)
	if electedTime == block.Header.Time {
		//because genesis block time is 0, 1 height block become new round, so change only electedtime and skip othe process
		if block.Header.Height == 1 {
			state.ElectedTime = electedTime - cs.period
		} else {
			miners, err := state.GetNewRoundMiners(block.Header.Time, cs.totalMiners)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{}).Panic(err)
				return err
			}
			state.MinersHash, err = state.PutMiners(miners)
			state.ElectedTime = block.Header.Time

			iter, err := state.Voter.Iterator(nil)
			if err != nil {
				if err != trie.ErrNotFound {
					log.CLog().WithFields(logrus.Fields{}).Panic(err)
					return err
				}
			} else {
				exist, _ := iter.Next()
				for exist {
					account := accs.GetAccount(common.BytesToAddress(iter.Key()))
					account.CalcSetTotalPeggedStake()
					accs.PutAccount(account)
					exist, err = iter.Next()
				}
			}
			//reset voter if this round is new
			state.Voter, err = trie.NewTrie(nil, cs.bc.Storage, false)
		}
	}
	err = state.Put(block.Header.Height, state.ElectedTime, state.MinersHash)
	if err != nil {
		return err
	}
	return nil
}



func (cs *Dpos) UpdateLIB() {
	bc := cs.bc
	block := bc.Tail()
	//FIXME: consider timestamp, changed minerGroup
	miners := make(map[common.Address]bool)
	turn := 1
	for bc.Lib().Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		if turn == int(cs.totalMiners) {
			if len(miners) == int(cs.totalMiners) {
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

func (c *Dpos) ConsensusType() string {
	return "DPOS"
}

func (cs *Dpos) LoadState(block *core.Block) (state core.ConsensusState, err error) {
	bc := cs.bc

	state, err = NewInitState(block.Header.ConsensusHash, block.Header.Height, bc.Storage)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (cs *Dpos) MakeGenesisBlock(block *core.Block, voters []*core.Account) (err error) {
	bc := cs.bc

	state, err := NewInitState(common.Hash{}, 0, bc.Storage)
	if err != nil {
		return err
	}

	//TODO: who voter?
	for _, v := range voters {
		state.Stake(v.Address, v.Address, v.Balance)
	}
	miners, err := state.GetNewRoundMiners(block.Header.Time, cs.totalMiners)
	if err != nil {
		return err
	}
	state.MinersHash, err = state.PutMiners(miners)
	state.ElectedTime = block.Header.Time
	state.Put(block.Header.Height, state.ElectedTime, state.MinersHash)

	block.SetConsensusState(state)
	block.Header.ConsensusHash = state.RootHash()
	bc.GenesisBlock = block
	bc.GenesisBlock.MakeHash()
	return nil
}

func (cs *Dpos) AddBlockChain(bc *core.BlockChain) {
	cs.bc = bc
}
