package pow

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/common"
	"github.com/sirupsen/logrus"

	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
)

type Pow struct {
	bc           *core.BlockChain
	coinbase     common.Address
	wallet       *account.Wallet
	enableMining bool
	streamPool   *net.PeerStreamPool

	period      uint64
	round       uint64
	totalMiners uint64
}

func NewPow(streamPool *net.PeerStreamPool, period, round, totalMiners uint64) *Pow {
	return &Pow{streamPool: streamPool,
		period:      period,
		round:       round,
		totalMiners: totalMiners}
}

func (cs *Pow) SetupMining(address common.Address, wallet *account.Wallet) {
	cs.enableMining = true
	cs.coinbase = address
	cs.wallet = wallet
}

//copied from ethereum
var (
	two256                 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	expDiffPeriod          = big.NewInt(100000)
	DifficultyBoundDivisor = big.NewInt(2048)   // The bound divisor of the difficulty, used in the update calculations.
	GenesisDifficulty      = big.NewInt(131072) // Difficulty of the Genesis block.
	MinimumDifficulty      = big.NewInt(131072) // The minimum that the difficulty may ever be.
	DurationLimit          = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
)

//calcDifficultyFrontier in ethereum
func calcDifficulty(time uint64, parent *core.Header) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisor)
	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.Set(new(big.Int).SetUint64(parent.Time))

	if bigTime.Sub(bigTime, bigParentTime).Cmp(DurationLimit) < 0 {
		diff.Add(parent.Difficulty, adjust)
	} else {
		diff.Sub(parent.Difficulty, adjust)
	}
	if diff.Cmp(MinimumDifficulty) < 0 {
		diff.Set(MinimumDifficulty)
	}

	big1 := new(big.Int).SetUint64(1)
	big2 := new(big.Int).SetUint64(2)

	periodCount := new(big.Int).Add(new(big.Int).SetUint64(parent.Height), big1)
	periodCount.Div(periodCount, expDiffPeriod)
	if periodCount.Cmp(big1) > 0 {
		// diff = diff + 2^(periodCount - 2)
		expDiff := periodCount.Sub(periodCount, big2)
		expDiff.Exp(big2, expDiff, nil)
		diff.Add(diff, expDiff)
		diff = math.BigMax(diff, params.MinimumDifficulty)
	}
	return diff
}

func makeHashNonce(hash []byte, nonce uint64) []byte {
	// Combine header+nonce into a 64 byte seed
	seed := make([]byte, 140)
	copy(seed, hash)
	binary.LittleEndian.PutUint64(seed[32:], nonce)

	return crypto.Keccak256(seed)
}

/*
	header.Difficulty,
*/
func (cs *Pow) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	block, err := bc.NewBlockFromTail()
	if err != nil {
		log.CLog().Warning(fmt.Sprintf("%+v", err))
	}
	block.Header.Time = now
	block.Header.Difficulty = calcDifficulty(now, bc.Tail.Header)

	//parent := bc.GetBlockByHash(bc.Tail.Header.ParentHash)

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
	//TODO: abort when receving new block with same height
	target := new(big.Int).Div(two256, block.Header.Difficulty)
	for i := 0; i < 10000000; i++ {
		seed := rand.Int63()
		result := makeHashNonce(common.HashToBytes(block.Hash()), uint64(seed+int64(i)))
		if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
			block.Header.Nonce = uint64(seed + int64(i))
			fmt.Println(i, "break")
			fmt.Println("target:", len(fmt.Sprintf("%s", target)), target)
			fmt.Println("result:", len(fmt.Sprintf("%s", new(big.Int).SetBytes(result))), new(big.Int).SetBytes(result))
			return block
		}
	}
	return nil
}

func (cs *Pow) Start() {
	if cs.enableMining {
		go cs.loop()
	}
}

func (cs *Pow) loop() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			block := cs.MakeBlock(uint64(now.Unix()))
			if block != nil {
				// sig, err := cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
				// if err != nil {
				// 	log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
				// }
				// block.SignWithSignature(sig)
				cs.bc.PutBlockByCoinbase(block)
				cs.bc.Consensus.UpdateLIB()
				cs.bc.RemoveOrphanBlock()
				message, _ := net.NewRLPMessage(net.MsgNewBlock, block.BaseBlock)
				cs.streamPool.BroadcastMessage(&message)
			}
		}
	}
}

func (cs *Pow) Verify(block *core.Block) (err error) {

	//TODO: check Difficulty
	//block.Header.Difficulty = calcDifficulty(block.Header.Time, parent.Header)
	// > 0
	// if header.Difficulty.Sign() <= 0 {
	// 	return errInvalidDifficulty
	// }

	target := new(big.Int).Div(two256, block.Header.Difficulty)
	result := makeHashNonce(common.HashToBytes(block.Hash()), block.Header.Nonce)
	if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
		return nil
	}
	return errors.New("not solved")
}

// not use this at GenesisBlock
func (cs *Pow) SaveState(block *core.Block) (err error) {
	return nil
}

//----------    Consensus  ----------------//

//TODO: How to use lib
func (cs *Pow) UpdateLIB() {
	return
}

func (c *Pow) ConsensusType() string {
	return "POW"
}

//TODO: What result to return
func (cs *Pow) LoadState(block *core.Block) (state core.ConsensusState, err error) {
	return nil, nil
}

func (cs *Pow) MakeGenesisBlock(block *core.Block, voters []*core.Account) (err error) {
	bc := cs.bc
	block.Header.Difficulty = GenesisDifficulty
	bc.GenesisBlock = block
	bc.GenesisBlock.MakeHash()
	return nil
}

func (cs *Pow) AddBlockChain(bc *core.BlockChain) {
	cs.bc = bc
}
