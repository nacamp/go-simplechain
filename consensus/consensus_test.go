package consensus_test

import (
	"math/big"
	"testing"

	"github.com/najimmy/go-simplechain/cmd"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/tests"

	"github.com/stretchr/testify/assert"

	"github.com/najimmy/go-simplechain/consensus"
	"github.com/najimmy/go-simplechain/core"
)

func TestMakeBlock(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()

	storage01, _ := storage.NewMemoryStorage()
	// storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		remoteBc := core.NewBlockChain(storage1)
		remoteBc.Setup(cs, voters)

		var block *core.Block
		if cs.ConsensusType() == "DPOS" {
			cs.(*consensus.Dpos).Setup(common.HexToAddress(tests.Addr0), common.FromHex(tests.Keystore[tests.Addr0]))
			block = cs.(*consensus.Dpos).MakeBlock(uint64(1)) // minerGroup[0]
		} else {
			cs.(*consensus.Poa).Setup(common.HexToAddress(tests.Addr0), common.FromHex(tests.Keystore[tests.Addr0]), 3)
			block = cs.(*consensus.Poa).MakeBlock(uint64(9 + 1)) // minerGroup[0]
		}
		assert.NotNil(t, block, "")
		assert.NotEqual(t, block.Header.AccountHash, remoteBc.GenesisBlock.Header.AccountHash, "")
		assert.Equal(t, block.Header.VoterHash, remoteBc.GenesisBlock.Header.VoterHash, "")
		assert.Equal(t, block.Header.MinerHash, remoteBc.GenesisBlock.Header.MinerHash, "")
		assert.Equal(t, block.Header.TransactionHash, remoteBc.GenesisBlock.Header.TransactionHash, "")
	}
}

/*
At N+3, LIB set N1
N+1		N+2		N+3
addr0   addr1   addr2
*/
func TestUpdateLIB1(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()

	storage01, _ := storage.NewMemoryStorage()
	// storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs, voters)

		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block1 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block1)
		cs.SaveMiners(block1)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block2 := tests.MakeBlock(bc, block1, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block2)
		cs.SaveMiners(block2)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block3 := tests.MakeBlock(bc, block2, tests.Addr2, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block3)
		cs.SaveMiners(block3)
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")
		cs.UpdateLIB()
		assert.Equal(t, block1.Hash(), bc.Lib.Hash(), "")
	}
}

/*
At N+5, LIB set N+3
N+1		N+2		N+3     N+4		N+5
addr0	addr1	addr2
				addr0	addr1	addr2
*/
func TestUpdateLIB2(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()
	storage01, _ := storage.NewMemoryStorage()
	// storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs, voters)

		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block1 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block1)
		cs.SaveMiners(block1)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block2 := tests.MakeBlock(bc, block1, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block2)
		cs.SaveMiners(block2)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block3 := tests.MakeBlock(bc, block2, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block3)
		cs.SaveMiners(block3)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block4 := tests.MakeBlock(bc, block3, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block4)
		cs.SaveMiners(block4)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block5 := tests.MakeBlock(bc, block4, tests.Addr2, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block5)
		cs.SaveMiners(block5)
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")
		cs.UpdateLIB()
		assert.Equal(t, block3.Hash(), bc.Lib.Hash(), "")
	}
}

/*
At N+5, LIB set N+5
At N+4, LIB set N+2
At N+3, LIB set N+1
N+1		N+2		N+3     N+4		N+5
addr0	addr1	addr2   addr0	addr2
				addr0
*/
func TestUpdateLIB3(t *testing.T) {

	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()
	storage01, _ := storage.NewMemoryStorage()
	storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs, voters)

		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block1 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block1)
		cs.SaveMiners(block1)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block2 := tests.MakeBlock(bc, block1, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block2)
		cs.SaveMiners(block2)
		cs.UpdateLIB()
		assert.Equal(t, bc.GenesisBlock.Hash(), bc.Lib.Hash(), "")

		block3 := tests.MakeBlock(bc, block2, tests.Addr2, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block3)
		cs.SaveMiners(block3)
		cs.UpdateLIB()
		assert.Equal(t, block1.Hash(), bc.Lib.Hash(), "")

		block4 := tests.MakeBlock(bc, block3, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block4)
		cs.SaveMiners(block4)
		cs.UpdateLIB()
		assert.Equal(t, block2.Hash(), bc.Lib.Hash(), "")

		block5 := tests.MakeBlock(bc, block4, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block5)
		cs.SaveMiners(block5)
		cs.UpdateLIB()
		assert.Equal(t, block3.Hash(), bc.Lib.Hash(), "")

		//test LoadLibFromStorage with same storage
		var cs2 core.Consensus
		if cs.ConsensusType() == "DPOS" {
			cs2 = consensus.NewDpos(nil)
			bc2 := core.NewBlockChain(storage1)
			bc2.Setup(cs2, voters)
			assert.Equal(t, bc.Lib.Hash(), bc2.Lib.Hash(), "")
			//check status loading
			assert.NotNil(t, bc2.Lib.VoterState, "")

			//test LoadTailFromStorage with same storage
			assert.Equal(t, bc.Tail.Hash(), block5.Hash(), "")
			//check status loading
			assert.NotNil(t, bc2.Tail.VoterState, "")
		} else {
			cs2 = consensus.NewPoa(nil, storage02)
			bc2 := core.NewBlockChain(storage1)
			bc2.Setup(cs2, voters)
			assert.Equal(t, bc.Lib.Hash(), bc2.Lib.Hash(), "")

			//test LoadTailFromStorage with same storage
			assert.Equal(t, bc.Tail.Hash(), block5.Hash(), "")
		}

	}
}
