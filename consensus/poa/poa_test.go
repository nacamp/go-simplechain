package poa

import (
	"fmt"
	"testing"
	"time"

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

	cs := NewPoa(net.NewPeerStreamPool(), mstrg)
	wallet := account.NewWallet(config.KeystoreFile)
	wallet.Load()
	err = wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
	if err != nil {
		log.CLog().Fatal(err)
	}

	cs.Setup(common.HexToAddress(config.MinerAddress), wallet, 3)
	bc := core.NewBlockChain(mstrg)

	//test MakeGenesisBlock in Setup
	bc.Setup(cs, voters)
	state, _ := NewInitState(cs.bc.GenesisBlock.ConsensusState().RootHash(), 0, mstrg)
	fmt.Println("0>>>>", common.HashToHex(cs.bc.GenesisBlock.ConsensusState().RootHash()))
	_, err = state.Signer.Get(common.FromHex(tests.Addr0))
	assert.NoError(t, err)
	_, err = state.Signer.Get(common.FromHex(tests.Addr1))
	assert.NoError(t, err)
	_, err = state.Signer.Get(common.FromHex(tests.Addr2))
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

	//TODO:  test UpdateLIB() with dpos at blockchain_test
}
