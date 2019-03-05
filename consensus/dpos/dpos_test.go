package dpos

import (
	"fmt"
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
	"github.com/sirupsen/logrus"
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
}

type DposMiner struct {
	Turn int
	Cs   *Dpos
	Bc   *core.BlockChain
}

func NewDposMiner(index int) *DposMiner {
	var err error
	//config
	config := tests.NewConfig(index)
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
	bc.Setup(cs, voters)


	tester := new(DposMiner)
	tester.Cs = cs
	tester.Bc = bc
	tester.Turn = findTurn(cs.coinbase, 0)
	return tester
}

func (m *DposMiner) MakeBlock(time int) *core.Block {
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

func findTurn(address common.Address, time int64) int {
	tempMiners := make([]common.Address, 0)
	tempMiners = append(tempMiners, tests.Address0)
	tempMiners = append(tempMiners, tests.Address1)
	tempMiners = append(tempMiners, tests.Address2)
	shuffle(tempMiners, time)
	for i, v := range tempMiners {
		if v == address {
			return i
		}
	}
	return -1

}

func TestNewRound(t *testing.T) {
	miner3 := NewDposMiner(0)
	miner1 := NewDposMiner(1)
	miner2 := NewDposMiner(2)
	bc1 := miner1.Bc
	bc2 := miner2.Bc
	bc3 := miner3.Bc

	turn := findTurn(tests.Address0, 0)
	var err error

	//miner1 and miner2 mine 1 block before new round test because electedTime is changed  at first block
	block1 := miner1.MakeBlock(27 + 3*(0+3*0))
	err = bc1.PutBlock(block1)
	assert.NoError(t, err)
	block1 = miner2.MakeBlock(27 + 3*(1+3*0))
	err = bc2.PutBlock(block1)
	assert.NoError(t, err)

	block1 = miner3.MakeBlock(27 + 3*(turn+3*0))
	err = bc3.PutBlock(block1)
	assert.NoError(t, err)

	block2 := miner3.MakeBlock(27 + 3*(turn+3*1))
	err = bc3.PutBlock(block2)
	assert.NoError(t, err)

	block3 := miner3.MakeBlock(27 + 3*(turn+3*2))
	err = bc3.PutBlock(block3)
	assert.NoError(t, err)

	//new round
	for i := 3; i < 10; i++ {
		newTurn := findTurn(tests.Address0, int64(27+3*(turn+3*i)))
		if turn != newTurn {
			fmt.Println("newTurn : ", newTurn)
			block := miner3.MakeBlock(27 + 3*(turn+3*i))
			assert.Nil(t, block)
			if turn == findTurn(tests.Address1, int64(27+3*(turn+3*i))) {
				block := miner1.MakeBlock(27 + 3*(turn+3*i))
				err = bc1.PutBlock(block)
				assert.NoError(t, err)
				fmt.Println(tests.Addr1, " mined")
			} else if turn == findTurn(tests.Address2, int64(27+3*(turn+3*i))) {
				block := miner2.MakeBlock(27 + 3*(turn+3*i))
				err = bc2.PutBlock(block)
				assert.NoError(t, err)
				fmt.Println(tests.Addr2, " mined")
			}
			break
		}
	}

}

/*
At N+3, LIB set N1
N+1		N+2		N+3
miner1  miner2   miner3
*/
func TestUpdateLIBN1(t *testing.T) {
	miner3 := NewDposMiner(0)
	miner1 := NewDposMiner(1)
	miner2 := NewDposMiner(2)
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
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block2 := miner2.MakeBlock(27 + 3*1)
	err = bc1.PutBlock(block2)
	assert.NoError(t, err)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)
	err = bc3.PutBlock(block2)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block3 := miner3.MakeBlock(27 + 3*2)
	err = bc1.PutBlock(block3)
	assert.NoError(t, err)
	err = bc2.PutBlock(block3)
	assert.NoError(t, err)
	err = bc3.PutBlock(block3)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, block1.Hash(), bc1.Lib.Hash(), "")

}

/*
At N+5, LIB set N+3
N+1		N+2		N+3     N+4		N+5
miner1	miner2	miner3
				miner1	miner2	miner3
*/
func TestUpdateLIBN3(t *testing.T) {
	miner3 := NewDposMiner(0)
	miner1 := NewDposMiner(1)
	miner2 := NewDposMiner(2)
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
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block2 := miner2.MakeBlock(27 + 3*1)
	err = bc1.PutBlock(block2)
	assert.NoError(t, err)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)
	err = bc3.PutBlock(block2)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block33 := miner3.MakeBlock(27 + 3*2)
	block31 := miner1.MakeBlock(27 + 3*3)
	err = bc1.PutBlock(block31)
	assert.NoError(t, err)
	err = bc2.PutBlock(block31)
	assert.NoError(t, err)
	err = bc3.PutBlock(block33)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")
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
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block5 := miner3.MakeBlock(27 + 3*5)
	err = bc1.PutBlock(block5)
	assert.NoError(t, err)
	err = bc2.PutBlock(block5)
	assert.NoError(t, err)
	err = bc3.PutBlock(block5)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, block31.Hash(), bc1.Lib.Hash(), "")

}

/*
At N+6, LIB set N+4
N+1		N+2		N+3      N+4	N+5     N+6
miner1	miner2	miner3   miner2	miner3  miner1
				miner1
*/
func TestUpdateLIB3(t *testing.T) {
	miner3 := NewDposMiner(0)
	miner1 := NewDposMiner(1)
	miner2 := NewDposMiner(2)
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
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block2 := miner2.MakeBlock(27 + 3*1)
	err = bc1.PutBlock(block2)
	assert.NoError(t, err)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)
	err = bc3.PutBlock(block2)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block33 := miner3.MakeBlock(27 + 3*2)
	block31 := miner1.MakeBlock(27 + 3*3)
	err = bc1.PutBlock(block31)
	assert.NoError(t, err)
	err = bc2.PutBlock(block33)
	assert.NoError(t, err)
	err = bc3.PutBlock(block33)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")
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
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block5 := miner3.MakeBlock(27 + 3*5)
	err = bc1.PutBlock(block5)
	assert.NoError(t, err)
	err = bc2.PutBlock(block5)
	assert.NoError(t, err)
	err = bc3.PutBlock(block5)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, bc1.GenesisBlock.Hash(), bc1.Lib.Hash(), "")

	block6 := miner1.MakeBlock(27 + 3*6)
	err = bc1.PutBlock(block6)
	assert.NoError(t, err)
	err = bc2.PutBlock(block6)
	assert.NoError(t, err)
	err = bc3.PutBlock(block6)
	assert.NoError(t, err)
	bc1.Consensus.UpdateLIB()
	assert.Equal(t, block4.Hash(), bc1.Lib.Hash(), "")
}

/*
     N0  LIB
   /   \
N1       N2
|        |
N4		 N3
|        |
N5       N6
|
N7
*/
// At PutBlockByCoinbase SetTail call RebuildBlockHeight
func TestRebuildBlockHeight(t *testing.T) {
	miner3 := NewDposMiner(0)
	miner1 := NewDposMiner(1)
	miner2 := NewDposMiner(2)
	bc1 := miner1.Bc
	bc2 := miner2.Bc
	bc3 := miner3.Bc
	var err error
	var b *core.Block

	// block1 := miner2.MakeBlock(27 + 3*1)
	// err = bc2.PutBlock(block1)
	// assert.NoError(t, err)

	//mining 1~3 height by miner2
	block2 := miner2.MakeBlock(27 + 3*4)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)

	block3 := miner2.MakeBlock(27 + 3*7)
	err = bc2.PutBlock(block3)
	assert.NoError(t, err)

	block6 := miner2.MakeBlock(27 + 3*28) // shuffle because of new round ( 3*10 order is not same)
	err = bc2.PutBlock(block6)
	assert.NoError(t, err)

	//mining 1~4 height by miner3
	block1 := miner3.MakeBlock(27 + 3*2)
	err = bc3.PutBlock(block1)
	assert.NoError(t, err)

	block4 := miner3.MakeBlock(27 + 3*5)
	err = bc3.PutBlock(block4)
	assert.NoError(t, err)

	block5 := miner3.MakeBlock(27 + 3*8) // shuffle because of new round ( 3*10 order is not same)
	err = bc3.PutBlock(block5)
	assert.NoError(t, err)

	block7 := miner3.MakeBlock(27 + 3*11) // shuffle because of new round ( 3*10 order is not same)
	err = bc3.PutBlock(block7)
	assert.NoError(t, err)

	//1,4,5 from miner3
	err = bc1.PutBlockIfParentExist(block1)
	b = bc1.GetBlockByHeight(1)
	assert.Equal(t, block1.Hash(), b.Hash())

	err = bc1.PutBlockIfParentExist(block4)
	b = bc1.GetBlockByHeight(2)
	assert.Equal(t, block4.Hash(), b.Hash())

	err = bc1.PutBlockIfParentExist(block5)
	b = bc1.GetBlockByHeight(3)
	assert.Equal(t, block5.Hash(), b.Hash())

	//2,3,6 from miner2
	err = bc1.PutBlockIfParentExist(block2)
	b = bc1.GetBlockByHeight(1)
	assert.Equal(t, block2.Hash(), b.Hash())

	err = bc1.PutBlockIfParentExist(block3)
	b = bc1.GetBlockByHeight(2)
	assert.Equal(t, block3.Hash(), b.Hash())

	err = bc1.PutBlockIfParentExist(block6)
	b = bc1.GetBlockByHeight(3)
	assert.Equal(t, block6.Hash(), b.Hash())

	//set lib block2 , height 1
	bc1.SetLib(block2)

	//7 from miner3
	err = bc1.PutBlockIfParentExist(block7)
	b = bc1.GetBlockByHeight(4)
	assert.Equal(t, block7.Hash(), b.Hash())

	//blocchain changed from block7' parents to Lib's height
	// front of Lib not changed
	b = bc1.GetBlockByHeight(1)
	assert.Equal(t, block2.Hash(), b.Hash())

	// behind Lib, changed
	b = bc1.GetBlockByHeight(2)
	assert.Equal(t, block4.Hash(), b.Hash())
	b = bc1.GetBlockByHeight(3)
	assert.Equal(t, block5.Hash(), b.Hash())
}

/*
	N0
	|
	N1
   /    \
N2        N3
|        /  \
N6(LIB)	N4
|
N7
|
N8
*/
/*
	err = bc2.PutBlockIfParentExist(block7)
	assert.NoError(t, err)
*/
func TestRemoveOrphanBlock(t *testing.T) {
	miner3 := NewDposMiner(0)
	miner1 := NewDposMiner(1)
	miner2 := NewDposMiner(2)
	bc1 := miner1.Bc
	bc2 := miner2.Bc
	bc3 := miner3.Bc
	var err error
	var b *core.Block

	block1 := miner2.MakeBlock(27 + 3*1)
	err = bc2.PutBlock(block1)
	assert.NoError(t, err)

	block2 := miner2.MakeBlock(27 + 3*4)
	err = bc2.PutBlock(block2)
	assert.NoError(t, err)

	err = bc3.PutBlockIfParentExist(block1)
	block3 := miner3.MakeBlock(27 + 3*2)
	err = bc3.PutBlock(block3)
	assert.NoError(t, err)

	block4 := miner3.MakeBlock(27 + 3*5)
	err = bc3.PutBlock(block4)
	assert.NoError(t, err)

	// block5 := miner2.MakeBlock(27 + 3*4)
	// err = bc2.PutBlock(block5)
	// assert.NoError(t, err)

	block6 := miner2.MakeBlock(27 + 3*7)
	err = bc2.PutBlock(block6)
	assert.NoError(t, err)

	block7 := miner2.MakeBlock(27 + 3*28)
	err = bc2.PutBlock(block7)
	assert.NoError(t, err)

	block8 := miner2.MakeBlock(27 + 3*31)
	err = bc2.PutBlock(block8)
	assert.NoError(t, err)

	// //1,4,5 from miner3
	err = bc1.PutBlockIfParentExist(block1)
	assert.NoError(t, err)
	err = bc1.PutBlockIfParentExist(block2)
	assert.NoError(t, err)
	err = bc1.PutBlockIfParentExist(block3)
	assert.NoError(t, err)
	err = bc1.PutBlockIfParentExist(block4)
	assert.NoError(t, err)
	err = bc1.PutBlockIfParentExist(block6)
	assert.NoError(t, err)
	err = bc1.PutBlockIfParentExist(block7)
	assert.NoError(t, err)
	err = bc1.PutBlockIfParentExist(block8)
	assert.NoError(t, err)

	bc1.SetLib(block6)
	bc1.SetTail(block6)

	assert.Equal(t, bc1.TxPool.Len(), 0, "")
	bc1.RemoveOrphanBlock()
	b = bc1.GetBlockByHash(block3.Hash())
	assert.Nil(t, b, "")

	b = bc1.GetBlockByHash(block4.Hash())
	assert.Nil(t, b, "")

	// N3 same tx,  N4,N5 different tx
	// assert.Equal(t, bc1.TxPool.Len(), 2, "")
}