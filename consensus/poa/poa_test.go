package poa

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/tests"
)

func TestPoa(t *testing.T) {
	var err error
	var block *core.Block
	//config
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	mstrg, _ := storage.NewMemoryStorage()

	cs := NewPoa(net.NewPeerStreamPool(), config.Consensus.Period)
	wallet := account.NewWallet(config.KeystoreFile)
	wallet.Load()
	err = wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
	if err != nil {
		log.CLog().Fatal(err)
	}

	cs.SetupMining(common.HexToAddress(config.MinerAddress), wallet)
	bc := core.NewBlockChain(mstrg, common.HexToAddress(config.Coinbase), uint64(config.MiningReward))

	//test MakeGenesisBlock in Setup
	bc.Setup(cs, voters[:3])
	state, _ := NewInitState(cs.bc.GenesisBlock.ConsensusState().RootHash(), 0, mstrg)
	fmt.Println("0>>>>", common.HashToHex(cs.bc.GenesisBlock.ConsensusState().RootHash()))
	_, err = state.Signer.Get(common.FromHex(tests.AddressHex0))
	assert.NoError(t, err)
	_, err = state.Signer.Get(common.FromHex(tests.AddressHex1))
	assert.NoError(t, err)
	_, err = state.Signer.Get(common.FromHex(tests.AddressHex2))
	assert.NoError(t, err)

	//test MakeBlock
	signers, err := state.GetMiners()
	assert.Equal(t, signers[0], cs.coinbase)
	block = cs.MakeBlock(3 * 4) // 3*4=>1, 3*5=>2, , 3*6=>0,
	assert.Nil(t, block)
	block = cs.MakeBlock(3 * 5) // 3*4=>1, 3*5=>2, , 3*6=>0,
	assert.Nil(t, block)

	block = cs.MakeBlock(3 * 6) // 3*4=>1, 3*5=>2, , 3*6=>0,
	sig, err := cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
	block.SignWithSignature(sig)
	err = bc.PutBlock(block)
	assert.Equal(t, block.ConsensusState().RootHash(), block.Header.ConsensusHash)
	fmt.Println("1>>>>", common.HashToHex(block.ConsensusState().RootHash()))
	assert.NoError(t, err)

	block = cs.MakeBlock(3 * 9) // 3*4=>1, 3*5=>2, , 3*6=>0,
	sig, err = cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
	block.SignWithSignature(sig)
	err = bc.PutBlock(block)
	assert.Equal(t, block.ConsensusState().RootHash(), block.Header.ConsensusHash)
	assert.NoError(t, err)
	fmt.Println("2>>>>", common.HashToHex(block.ConsensusState().RootHash()))

	//test Verify
	assert.NoError(t, cs.Verify(block))
	block.Header.Time = 3 * 4
	assert.Error(t, cs.Verify(block))

	//test getMinerSize
	minerSize, _ := cs.getMinerSize(block)
	assert.Equal(t, 3, minerSize)
}

type PoaMiner struct {
	Turn int
	Cs   *Poa
	Bc   *core.BlockChain
}

func NewPoaMiner(index int) *PoaMiner {
	var err error
	//config
	config := tests.NewConfig(index)
	voters := cmd.MakeVoterAccountsFromConfig(config)
	mstrg, _ := storage.NewMemoryStorage()

	cs := NewPoa(net.NewPeerStreamPool(), config.Consensus.Period)
	wallet := account.NewWallet(config.KeystoreFile)
	wallet.Load()
	err = wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
	if err != nil {
		log.CLog().Fatal(err)
	}

	cs.SetupMining(common.HexToAddress(config.MinerAddress), wallet)
	bc := core.NewBlockChain(mstrg, common.HexToAddress(config.Coinbase), uint64(config.MiningReward))
	bc.Setup(cs, voters[:3])

	tester := new(PoaMiner)
	tester.Cs = cs
	tester.Bc = bc
	go func() {
		for {
			select {
			case <-bc.LibCh:
			}
		}
	}()
	return tester
}

func (m *PoaMiner) MakeBlock(time int) *core.Block {
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
		return block
	}
	return nil

}

func TestSendMoneyTransaction(t *testing.T) {
	miner1 := NewPoaMiner(0)
	// miner2 := NewPoaMiner(1)
	// miner3 := NewPoaMiner(2)
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

func TestVoteTransaction(t *testing.T) {
	miner1 := NewPoaMiner(0)
	miner2 := NewPoaMiner(1)
	miner3 := NewPoaMiner(2)
	bc1 := miner1.Bc
	bc2 := miner2.Bc
	bc3 := miner3.Bc
	// var err error
	_ = bc1
	_ = bc2
	_ = bc3

	var candidate = common.HexToAddress("0x11f75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2")
	voter := []common.Address{tests.Address0, tests.Address1, tests.Address2}
	signer := []*PoaMiner{miner1, miner2, miner3}
	nonces := []uint64{1, 1, 1}
	var tx *core.Transaction
	var block *core.Block
	//vote for joinning
	for i := 0; i < 2; i++ {
		payload := new(core.Payload)
		payload.Code = core.TxCVoteStake
		tx = core.NewTransactionPayload(voter[i], candidate, new(big.Int).SetUint64(0), nonces[i], payload)
		tx.MakeHash()
		sig, err := signer[i].Cs.wallet.SignHash(voter[i], tx.Hash[:])
		assert.NoError(t, err)
		tx.SignWithSignature(sig)
		signer[i].Bc.TxPool.Put(tx)

		block = signer[i].MakeBlock(3*3 + 3*i)
		for j := 0; j < 3; j++ {
			err = signer[j].Bc.PutBlock(block)
			assert.NoError(t, err)
		}

	}
	//After 2/3 of the signers vote for the candidate, the candidate becomes the signer in the next block
	state := block.ConsensusState().(*PoaState)
	signers, _ := state.GetMiners()
	assert.Equal(t, 3, len(signers))
	//changed signers in the next block
	block = signer[0].Cs.MakeBlock(uint64(4*3 + 4 + 3*0))
	state = block.ConsensusState().(*PoaState)
	signers, _ = state.GetMiners()
	assert.Equal(t, 4, len(signers))

	//vote for evicting
	nonces = []uint64{2, 2, 1}
	for i := 0; i < 3; i++ {
		payload := new(core.Payload)
		payload.Code = core.TxCVoteUnStake
		tx = core.NewTransactionPayload(voter[i], candidate, new(big.Int).SetUint64(0), nonces[i], payload)
		tx.MakeHash()
		sig, err := signer[i].Cs.wallet.SignHash(voter[i], tx.Hash[:])
		assert.NoError(t, err)
		tx.SignWithSignature(sig)
		signer[i].Bc.TxPool.Put(tx)

		block = signer[i].MakeBlock(4*3 + 4 + 3*i) //4 to skip the new signer
		for j := 0; j < 3; j++ {
			err = signer[j].Bc.PutBlock(block)
			assert.NoError(t, err)
		}

	}
	block = signer[0].Cs.MakeBlock(uint64(3*3 + 90 + 3*0))
	state = block.ConsensusState().(*PoaState)
	signers, _ = state.GetMiners()
	assert.Equal(t, 3, len(signers))
}

/*
At N+3, LIB set N1
N+1		N+2		N+3
miner1  miner2   miner3
*/
func TestUpdateLIBN1(t *testing.T) {
	miner1 := NewPoaMiner(0)
	miner2 := NewPoaMiner(1)
	miner3 := NewPoaMiner(2)
	bc1 := miner1.Bc
	bc2 := miner2.Bc
	bc3 := miner3.Bc
	var err error

	block1 := miner1.MakeBlock(27 + 3*0)
	err = bc1.PutBlock(block1)
	assert.NoError(t, err)
	err = bc2.PutBlock(block1)
	assert.NoError(t, err)
	err = bc3.PutBlock(block1)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block2 := miner2.MakeBlock(27 + 3*1)
	err = bc1.PutBlock(block2)
	assert.NoError(t, err)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)
	err = bc3.PutBlock(block2)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block3 := miner3.MakeBlock(27 + 3*2)
	err = bc1.PutBlock(block3)
	assert.NoError(t, err)
	err = bc2.PutBlock(block3)
	assert.NoError(t, err)
	err = bc3.PutBlock(block3)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, block1.Hash(), bc1.Lib().Hash(), "")

}

/*
At N+5, LIB set N+3
N+1		N+2		N+3     N+4		N+5
miner1	miner2	miner3
				miner1	miner2	miner3
*/
func TestUpdateLIBN3(t *testing.T) {
	miner1 := NewPoaMiner(0)
	miner2 := NewPoaMiner(1)
	miner3 := NewPoaMiner(2)
	bc1 := miner1.Bc
	bc2 := miner2.Bc
	bc3 := miner3.Bc
	var err error

	block1 := miner1.MakeBlock(27 + 3*0)
	err = bc1.PutBlock(block1)
	assert.NoError(t, err)
	err = bc2.PutBlock(block1)
	assert.NoError(t, err)
	err = bc3.PutBlock(block1)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block2 := miner2.MakeBlock(27 + 3*1)
	err = bc1.PutBlock(block2)
	assert.NoError(t, err)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)
	err = bc3.PutBlock(block2)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block33 := miner3.MakeBlock(27 + 3*2)
	block31 := miner1.MakeBlock(27 + 3*3)
	err = bc1.PutBlock(block31)
	assert.NoError(t, err)
	err = bc2.PutBlock(block31)
	assert.NoError(t, err)
	err = bc3.PutBlock(block33)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")
	//assert.Equal(t, block1.Hash(), bc1.Lib.Hash(), "")

	block4 := miner2.MakeBlock(27 + 3*4)
	err = bc1.PutBlock(block4)
	assert.NoError(t, err)
	err = bc2.PutBlock(block4)
	assert.NoError(t, err)
	err = bc3.PutBlockIfParentExist(block4)
	assert.NoError(t, err)
	err = bc3.PutBlockIfParentExist(block31) //receive missing block
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block5 := miner3.MakeBlock(27 + 3*5)
	err = bc1.PutBlock(block5)
	assert.NoError(t, err)
	err = bc2.PutBlock(block5)
	assert.NoError(t, err)
	err = bc3.PutBlock(block5)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, block31.Hash(), bc1.Lib().Hash(), "")

}

/*
At N+6, LIB set N+4
N+1		N+2		N+3      N+4	N+5     N+6
miner1	miner2	miner3   miner2	miner3  miner1
				miner1
*/
func TestUpdateLIB3(t *testing.T) {
	miner1 := NewPoaMiner(0)
	miner2 := NewPoaMiner(1)
	miner3 := NewPoaMiner(2)
	bc1 := miner1.Bc
	bc2 := miner2.Bc
	bc3 := miner3.Bc
	var err error

	block1 := miner1.MakeBlock(27 + 3*0)
	err = bc1.PutBlock(block1)
	assert.NoError(t, err)
	err = bc2.PutBlock(block1)
	assert.NoError(t, err)
	err = bc3.PutBlock(block1)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block2 := miner2.MakeBlock(27 + 3*1)
	err = bc1.PutBlock(block2)
	assert.NoError(t, err)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)
	err = bc3.PutBlock(block2)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block33 := miner3.MakeBlock(27 + 3*2)
	block31 := miner1.MakeBlock(27 + 3*3)
	err = bc1.PutBlock(block31)
	assert.NoError(t, err)
	err = bc2.PutBlock(block33)
	assert.NoError(t, err)
	err = bc3.PutBlock(block33)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")
	//assert.Equal(t, block1.Hash(), bc1.Lib.Hash(), "")

	block4 := miner2.MakeBlock(27 + 3*4)
	err = bc1.PutBlockIfParentExist(block4)
	assert.NoError(t, err)
	err = bc1.PutBlockIfParentExist(block33) //receive missing block
	assert.NoError(t, err)
	err = bc2.PutBlock(block4)
	assert.NoError(t, err)
	err = bc3.PutBlock(block4)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block5 := miner3.MakeBlock(27 + 3*5)
	err = bc1.PutBlock(block5)
	assert.NoError(t, err)
	err = bc2.PutBlock(block5)
	assert.NoError(t, err)
	err = bc3.PutBlock(block5)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib().Hash(), "")

	block6 := miner1.MakeBlock(27 + 3*6)
	err = bc1.PutBlock(block6)
	assert.NoError(t, err)
	err = bc2.PutBlock(block6)
	assert.NoError(t, err)
	err = bc3.PutBlock(block6)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, block4.Hash(), bc1.Lib().Hash(), "")
}
