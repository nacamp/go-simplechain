package pow

import (
	"math/big"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/tests"
)

func TestPow(t *testing.T) {
	var err error
	var block *core.Block
	//config
	config := tests.MakeConfig()
	mstrg, _ := storage.NewMemoryStorage()

	cs := NewPow(net.NewPeerStreamPool(), config.Consensus.Difficulty)
	wallet := account.NewWallet(config.KeystoreFile)
	wallet.Load()
	err = wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
	if err != nil {
		log.CLog().Fatal(err)
	}

	bc := core.NewBlockChain(mstrg, common.HexToAddress(config.Coinbase), uint64(config.MiningReward))

	//test MakeGenesisBlock in Setup
	bc.Setup(cs, []*core.Account{})

	block = cs.MakeBlock(3)
	assert.NotNil(t, block)
	assert.NoError(t, cs.Verify(block))

	//result := work([]byte{0x01}, uint64(18446744073709551615)) //uint64 max
}

type PowMiner struct {
	Cs *Pow
	Bc *core.BlockChain
}

func NewPowMiner(index int) *PowMiner {
	var err error
	//config
	config := tests.NewConfig(index)
	mstrg, _ := storage.NewMemoryStorage()

	cs := NewPow(net.NewPeerStreamPool(), config.Consensus.Difficulty)
	wallet := account.NewWallet(config.KeystoreFile)
	wallet.Load()
	err = wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
	if err != nil {
		log.CLog().Fatal(err)
	}

	cs.SetupMining(common.HexToAddress(config.MinerAddress), wallet)
	bc := core.NewBlockChain(mstrg, common.HexToAddress(config.Coinbase), uint64(config.MiningReward))
	bc.Setup(cs, []*core.Account{})

	tester := new(PowMiner)
	tester.Cs = cs
	tester.Bc = bc
	return tester
}

func (m *PowMiner) MakeBlock(time int) *core.Block {
	cs := m.Cs
	block := cs.MakeBlock(uint64(time))
	if block != nil {
		sig, err := cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Warning("SignHash")
			return nil
		}
		block.SignWithSignature(sig)
		cs.bc.PutBlockByCoinbase(block)
		cs.bc.Consensus.UpdateLIB()
		cs.bc.RemoveOrphanBlock()
		return block
	}
	return nil

}

func TestSendMoneyTransaction(t *testing.T) {
	miner1 := NewPowMiner(0)
	// miner2 := NewPowMiner(1)
	// miner3 := NewPowMiner(2)
	bc1 := miner1.Bc
	// bc2 := miner2.Bc
	// bc3 := miner3.Bc
	var err error

	block1 := miner1.MakeBlock(27 + 3*(0+3*0))
	err = bc1.PutBlock(block1)
	assert.NoError(t, err)

	//send money
	var tx *core.Transaction
	for i := 3; i >= 1; i-- {
		tx = core.NewTransaction(tests.Address0, tests.Address2, new(big.Int).SetUint64(5), uint64(i))
		tx.MakeHash()
		sig, err := miner1.Cs.wallet.SignHash(tests.Address0, tx.Hash[:])
		assert.NoError(t, err)
		tx.SignWithSignature(sig)
		bc1.TxPool.Put(tx)

	}

	//Put tx in descending order, 3, 2, 1
	tx = core.NewTransaction(tests.Address0, tests.Address2, new(big.Int).SetUint64(5), 10)
	tx.MakeHash()
	sig, err := miner1.Cs.wallet.SignHash(tests.Address0, tx.Hash[:])
	assert.NoError(t, err)
	tx.SignWithSignature(sig)
	bc1.TxPool.Put(tx)

	block2 := miner1.MakeBlock(9 + 3*(0+3*1))
	err = bc1.PutBlock(block2)
	assert.NoError(t, err)
	accs := block2.AccountState
	account2 := accs.GetAccount(tests.Address2)
	// nonce 3,2,1 is included, but 10 is not included
	assert.Equal(t, new(big.Int).SetUint64(15), account2.Balance)
}
