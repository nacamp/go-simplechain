package pow

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/sirupsen/logrus"

	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
)

type Pow struct {
	bc                *core.BlockChain
	coinbase          common.Address
	wallet            *account.Wallet
	enableMining      bool
	streamPool        *net.PeerStreamPool
	genesisDifficulty *big.Int //big.NewInt(5000000)
}

func NewPow(streamPool *net.PeerStreamPool, difficulty *big.Int) *Pow {
	return &Pow{streamPool: streamPool, genesisDifficulty: difficulty}
}

func (cs *Pow) SetupMining(address common.Address, wallet *account.Wallet) {
	cs.enableMining = true
	cs.coinbase = address
	cs.wallet = wallet
}

//code copied from ethereum >>>>>>>>>>
var (
	two256                 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	expDiffPeriod          = big.NewInt(100000)
	DifficultyBoundDivisor = big.NewInt(2048)   // The bound divisor of the difficulty, used in the update calculations.
	GenesisDifficulty      = big.NewInt(131072) // Difficulty of the Genesis block.
	MinimumDifficulty      = big.NewInt(131072) // The minimum that the difficulty may ever be.
	DurationLimit          = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.

	bigMinus99 = big.NewInt(-99)
	big1       = new(big.Int).SetUint64(1)
	big2       = new(big.Int).SetUint64(2)
	big10      = new(big.Int).SetUint64(10)
)

//calcDifficultyHomestead in ethereum
func calcDifficulty(time uint64, parent *core.Header) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).Set(new(big.Int).SetUint64(parent.Time))

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(MinimumDifficulty) < 0 {
		x.Set(MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(new(big.Int).SetUint64(parent.Height), big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}

func work(hash []byte, nonce uint64) []byte {
	newHash := make([]byte, 40)
	copy(newHash, hash)
	binary.LittleEndian.PutUint64(newHash[32:], nonce)
	return crypto.Sha3b256(newHash)
}

//code copied from ethereum <<<<<<<<<<<

func (cs *Pow) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	block, err := bc.NewBlockFromTail()
	if err != nil {
		log.CLog().Warning(fmt.Sprintf("%+v", err))
	}
	block.Header.Time = now
	block.Header.Difficulty = calcDifficulty(now, bc.Tail().Header)
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

	block.Header.AccountHash = block.AccountState.RootHash()
	block.Header.TransactionHash = block.TransactionState.RootHash()
	block.MakeHash()

	//mine
	seed := rand.Int63()
	target := new(big.Int).Div(two256, block.Header.Difficulty)
	inc := int64(0)
	for {
		if bc.Tail().Header.Height >= block.Header.Height {
			log.CLog().WithFields(logrus.Fields{
				"Height": block.Header.Height,
			}).Info("Other miner mined the block")
			return nil
		} else {
			result := work(common.HashToBytes(block.Hash()), uint64(seed+inc))
			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
				block.Header.Nonce = uint64(seed + inc)
				return block
			}
			inc++
		}
	}
	return nil
}

func (cs *Pow) loop() {
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

func (cs *Pow) Start() {
	if cs.enableMining {
		go cs.loop()
	}
}

func (cs *Pow) Verify(block *core.Block) (err error) {
	bc := cs.bc
	parent := bc.GetBlockByHash(block.Header.ParentHash)
	if parent == nil {
		return errors.New("Parent block is nil")
	}
	if block.Header.Difficulty.Cmp(calcDifficulty(block.Header.Time, parent.Header)) != 0 {
		return errors.New("Difficulty is not valid")
	}
	if block.Header.Difficulty.Cmp(new(big.Int).SetUint64(0)) <= 0 {
		return errors.New("Difficulty must be greater 0")
	}

	target := new(big.Int).Div(two256, block.Header.Difficulty)
	result := work(common.HashToBytes(block.Hash()), block.Header.Nonce)
	if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
		return nil
	}
	return errors.New("not solved")
}

//TODO: How to use lib
func (cs *Pow) UpdateLIB() {
	return
}

// not use this at GenesisBlock
func (cs *Pow) SaveState(block *core.Block) (err error) {
	return nil
}

func (c *Pow) ConsensusType() string {
	return "POW"
}

//TODO: What result to return
func (cs *Pow) LoadState(block *core.Block) (state core.ConsensusState, err error) {
	return &PowState{}, nil
}

func (cs *Pow) MakeGenesisBlock(block *core.Block, voters []*core.Account) (err error) {
	bc := cs.bc
	block.Header.Difficulty = cs.genesisDifficulty
	bc.GenesisBlock = block

	state := new(PowState)
	block.SetConsensusState(state)
	block.Header.ConsensusHash = state.RootHash()

	bc.GenesisBlock.MakeHash()
	return nil
}

func (cs *Pow) AddBlockChain(bc *core.BlockChain) {
	cs.bc = bc
}
