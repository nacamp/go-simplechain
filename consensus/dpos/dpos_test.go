package dpos

import (
	"reflect"
	"testing"
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/tests"
	"github.com/stretchr/testify/assert"
)

func TestDpos(t *testing.T) {
	var err error
	var block *core.Block
	//config
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	mstrg, _ := storage.NewMemoryStorage()

	cs := NewDpos(net.NewPeerStreamPool())
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
	miners, _ := state.GetMiners(state.MinersHash)
	assert.Equal(t, uint64(0), state.ElectedTime)
	assert.Equal(t, 3, len(miners))

	//test MakeBlock
	tempMiners := make([]common.Address, 0)
	tempMiners = append(tempMiners, tests.Address0)
	tempMiners = append(tempMiners, tests.Address1)
	tempMiners = append(tempMiners, tests.Address2)
	shuffle(tempMiners, 0)

	turn := 0
	wrongTurn := make([]int, 0)
	for i, v := range tempMiners {
		if v == cs.coinbase {
			turn = i
		} else {
			wrongTurn = append(wrongTurn, i)
		}
	}
	block = cs.MakeBlock(uint64(27 + 3*wrongTurn[0])) // turn := (now % 9) / 3
	assert.Nil(t, block)
	block = cs.MakeBlock(uint64(27 + 3*wrongTurn[1])) // turn := (now % 9) / 3
	assert.Nil(t, block)
	block = cs.MakeBlock(uint64(27 + 3*turn)) // turn := (now % 9) / 3
	assert.NotNil(t, block)

	//test Verify
	assert.NoError(t, cs.Verify(block))
	block.Header.Time = uint64(wrongTurn[0])
	assert.Error(t, cs.Verify(block))

	//test SaveState
	block.Header.Time = uint64(27 + turn*3)
	cs.SaveState(block)
	state2, _ := NewInitState(block.Header.ConsensusHash, 1, mstrg)
	miners2, _ := state2.GetMiners(state2.MinersHash)
	assert.True(t, reflect.DeepEqual(miners, miners2))
	//because genesis block time is 0, 1 height block become new round, so change only electedtime
	assert.Equal(t, uint64(27+turn*3-3), state2.ElectedTime) // ElectedTime = block.header.Time -3

	//TODO:  test UpdateLIB() with dpos at blockchain_test
	//TODO:  test new round case with dpos at blockchain_test
}
