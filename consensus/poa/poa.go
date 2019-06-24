package poa

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/nacamp/go-simplechain/account"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/sirupsen/logrus"
)

type Poa struct {
	mu sync.RWMutex
	bc *core.BlockChain
	// node         *net.Node
	coinbase     common.Address
	enableMining bool
	// Storage      storage.Storage
	period     uint64
	wallet     *account.Wallet
	streamPool *net.PeerStreamPool
}

func NewPoa(streamPool *net.PeerStreamPool, period uint64) *Poa {
	return &Poa{streamPool: streamPool, period: period}
}

func (cs *Poa) SetupMining(address common.Address, wallet *account.Wallet) {
	cs.enableMining = true
	cs.coinbase = address
	cs.wallet = wallet
}

func (cs *Poa) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	block, err := bc.NewBlockFromTail()
	if err != nil {
		log.CLog().Warning(fmt.Sprintf("%+v", err))
	}

	block.Header.Time = now
	state := block.ConsensusState().(*PoaState)
	state.RefreshSigner()
	miners, err := state.GetMiners()
	if len(miners) == 0 {
		log.CLog().WithFields(logrus.Fields{
			"Size": 0,
		}).Panic("Miner must be one more")
	}
	if err != nil {
		log.CLog().Warning(fmt.Sprintf("%+v", err))
	}
	turn := (now % (uint64(len(miners)) * cs.period)) / cs.period
	if miners[turn] == cs.coinbase {
		//parent := bc.GetBlockByHash(block.Header.ParentHash)
		parent := bc.Tail()

		//if (parent != nil) && (now-parent.Header.Time < ((uint64(len(miners)) * cs.Period) - 1)) { //(3 * 3)
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

		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		firstVote := true
		noncePool := make(map[common.Address][]*core.Transaction)
		for i := 0; i < bc.TxPool.Len(); i++ {
			tx := bc.TxPool.Pop()
			if tx == nil {
				break
			}
			fromAccount := accs.GetAccount(tx.From)
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
				// //use in future
				// bc.TxPool.Put(tx)
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

func (cs *Poa) loop() {
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
	state, err := cs.LoadState(parentBlock)
	if err != nil {
		return 0, err
	}
	poaState := state.(*PoaState)
	ms, err := poaState.GetMiners()
	if err != nil {
		return 0, err
	}
	minerSize = len(ms)
	return minerSize, nil
}

//----------    Consensus  ----------------//

func (cs *Poa) Start() {
	if cs.enableMining {
		go cs.loop()
	}
}

func (cs *Poa) UpdateLIB() {
	bc := cs.bc
	block := bc.Tail()
	miners := make(map[common.Address]bool)
	turn := 1
	if block.Header.Height == 0 {
		return
	}
	firstMinerSize, err := cs.getMinerSize(block)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
		return
	}
	if firstMinerSize < 3 {
		log.CLog().WithFields(logrus.Fields{
			"Size": firstMinerSize,
		}).Debug("At least 3 node are needed")
		return
	}
	for bc.Lib().Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		size, err := cs.getMinerSize(block)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
			return
		}
		if firstMinerSize != size {
			return
		}
		if turn == firstMinerSize {
			if len(miners) >= firstMinerSize*2/3+1 {
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

func (cs *Poa) MakeGenesisBlock(block *core.Block, voters []*core.Account) (err error) {
	bc := cs.bc
	bc.Signers = make([]common.Address, len(voters))
	for i, account := range voters {
		bc.Signers[i] = account.Address
	}

	state, err := NewInitState(common.Hash{}, 0, bc.Storage)
	if err != nil {
		return err
	}
	for _, v := range voters {
		state.Signer.Put(v.Address[:], []byte{})
	}
	state.Put(block.Header.Height)

	block.SetConsensusState(state)
	block.Header.ConsensusHash = state.RootHash()
	bc.GenesisBlock = block
	bc.GenesisBlock.MakeHash()
	return nil
}

func (cs *Poa) AddBlockChain(bc *core.BlockChain) {
	cs.bc = bc
}

func (cs *Poa) Verify(block *core.Block) error {
	state := block.ConsensusState().(*PoaState)
	miners, err := state.GetMiners()
	if err != nil {
		return err
	}
	index := (block.Header.Time % (uint64(len(miners)) * cs.period)) / cs.period
	if miners[index] != block.Header.Coinbase {
		return errors.New("This turn is not this miner's turn ")
	}
	return nil
}

func (cs *Poa) SaveState(block *core.Block) (err error) {
	state := block.ConsensusState().(*PoaState)
	//call state.RefreshSigner when loading(at MakeBlock, LoadState )
	err = state.Put(block.Header.Height)
	if err != nil {
		return err
	}
	return nil
}

func (cs *Poa) LoadState(block *core.Block) (state core.ConsensusState, err error) {
	bc := cs.bc

	state, err = NewInitState(block.Header.ConsensusHash, block.Header.Height, bc.Storage)
	if err != nil {
		return nil, err
	}
	state.(*PoaState).RefreshSigner()
	return state, nil
}
