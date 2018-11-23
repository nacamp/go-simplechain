package core_test

import (
	"math/big"
	"testing"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/consensus"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/tests"
	"github.com/stretchr/testify/assert"
)

func TestGenesisBlock(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	bc := core.NewBlockChain(dpos, storage)
	bc.MakeGenesisBlock(voters)

	assert.Equal(t, voters[0].Address, bc.GenesisBlock.Header.Coinbase, "")
	assert.Equal(t, bc.GenesisBlock.Header.SnapshotVoterTime, uint64(0), "")

	//check order by balance
	minerGroup, _, _ := bc.GenesisBlock.MinerState.GetMinerGroup(bc, bc.GenesisBlock)
	assert.Equal(t, voters[0].Address, minerGroup[0], "")
	assert.Equal(t, voters[2].Address, minerGroup[1], "")
	assert.Equal(t, voters[1].Address, minerGroup[2], "")
}

/*
func (bc *BlockChain) LoadBlockChainFromStorage() error {
	block, err := bc.GetBlockByHeight(0)
	if err != nil {
		return err
	}
	//status
	block.AccountState, _ = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	block.TransactionState, _ = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	block.VoterState, _ = NewAccountStateRootHash(block.Header.VoterHash, bc.Storage)
	block.MinerState, _ = bc.Consensus.NewMinerState(block.Header.MinerHash, bc.Storage)
	bc.GenesisBlock = block
	return nil

}
*/

func TestLoadBlockChainFromStorage(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	remoteBc := core.NewBlockChain(dpos, storage1)
	remoteBc.MakeGenesisBlock(voters)
	remoteBc.PutBlock(remoteBc.GenesisBlock)

	dpos2 := consensus.NewDpos()
	bc := core.NewBlockChain(dpos2, storage1)
	bc.LoadBlockChainFromStorage()

	assert.Equal(t, remoteBc.GenesisBlock.Hash(), bc.GenesisBlock.Hash(), "")
	assert.Equal(t, remoteBc.GenesisBlock.AccountState.RootHash(), bc.GenesisBlock.AccountState.RootHash(), "")
	assert.Equal(t, remoteBc.GenesisBlock.TransactionState.RootHash(), bc.GenesisBlock.TransactionState.RootHash(), "")
	assert.Equal(t, remoteBc.GenesisBlock.VoterState.RootHash(), bc.GenesisBlock.VoterState.RootHash(), "")
}

func TestSetup(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	remoteBc := core.NewBlockChain(dpos, storage1)
	remoteBc.Setup(voters)

	dpos2 := consensus.NewDpos()
	bc := core.NewBlockChain(dpos2, storage1)
	bc.LoadBlockChainFromStorage()

	assert.Equal(t, remoteBc.GenesisBlock.Hash(), bc.GenesisBlock.Hash(), "")
	assert.Equal(t, remoteBc.GenesisBlock.AccountState.RootHash(), bc.GenesisBlock.AccountState.RootHash(), "")
	assert.Equal(t, remoteBc.GenesisBlock.TransactionState.RootHash(), bc.GenesisBlock.TransactionState.RootHash(), "")
	assert.Equal(t, remoteBc.GenesisBlock.VoterState.RootHash(), bc.GenesisBlock.VoterState.RootHash(), "")
}

func TestStorage(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	bc := core.NewBlockChain(dpos, storage)
	bc.MakeGenesisBlock(voters)

	bc.PutBlock(bc.GenesisBlock)

	b1, _ := bc.GetBlockByHeight(0)
	assert.Equal(t, uint64(0), b1.Header.Height, "")
	assert.Equal(t, bc.GenesisBlock.Hash(), b1.Hash(), "")

	b2, _ := bc.GetBlockByHash(bc.GenesisBlock.Hash())
	assert.Equal(t, uint64(0), b2.Header.Height, "")
	assert.Equal(t, bc.GenesisBlock.Hash(), b2.Hash(), "")

	b3, _ := bc.GetBlockByHash(common.Hash{0x01})
	assert.Nil(t, b3, "")

	h := core.Header{}
	h.ParentHash = b1.Hash()
	block := core.Block{Header: &h}
	assert.Equal(t, true, bc.HasParentInBlockChain(&block), "")
	h.ParentHash.SetBytes([]byte{0x01})
	assert.Equal(t, false, bc.HasParentInBlockChain(&block), "")

}

func TestMakeBlockChain(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	remoteBc := core.NewBlockChain(dpos, storage1)
	remoteBc.MakeGenesisBlock(voters)

	remoteBc.PutBlockByCoinbase(remoteBc.GenesisBlock)
	block1 := tests.MakeBlock(remoteBc, remoteBc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block1)

	block2 := tests.MakeBlock(remoteBc, block1, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block2)

	block3 := tests.MakeBlock(remoteBc, block2, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block3)

	block4 := tests.MakeBlock(remoteBc, block3, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block4)

	storage2, _ := storage.NewMemoryStorage()

	dpos2 := consensus.NewDpos()
	bc := core.NewBlockChain(dpos2, storage2)
	//FIXME: how to test
	bc.TEST = true
	bc.MakeGenesisBlock(voters)
	bc.PutBlockByCoinbase(bc.GenesisBlock)
	// fmt.Printf("%v\n", bc.GenesisBlock.Hash())

	bc.PutBlockIfParentExist(block1)
	b, _ := bc.GetBlockByHash(block1.Hash())
	assert.Equal(t, block1.Hash(), b.Hash(), "")

	bc.PutBlockIfParentExist(block4)
	b, _ = bc.GetBlockByHash(block4.Hash())
	assert.Nil(t, b, "")

	bc.PutBlockIfParentExist(block3)
	b, _ = bc.GetBlockByHash(block3.Hash())
	assert.Nil(t, b, "")

	bc.PutBlockIfParentExist(block2)
	b, _ = bc.GetBlockByHash(block2.Hash())
	assert.NotNil(t, b, "")

	b, _ = bc.GetBlockByHash(block3.Hash())
	assert.NotNil(t, b, "")

	b, _ = bc.GetBlockByHash(block4.Hash())
	assert.NotNil(t, b, "")

}

func rlpEncode(block *core.Block) *core.Block {
	message, _ := net.NewRLPMessage(net.MSG_NEW_BLOCK, block)
	block2 := core.Block{}
	rlp.DecodeBytes(message.Payload, &block2)
	return &block2
}

func TestMakeBlockChainWhenRlpEncode(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	remoteBc := core.NewBlockChain(dpos, storage1)
	remoteBc.MakeGenesisBlock(voters)

	remoteBc.PutBlockByCoinbase(remoteBc.GenesisBlock)
	block1 := tests.MakeBlock(remoteBc, remoteBc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block1)

	block2 := tests.MakeBlock(remoteBc, block1, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block2)

	block3 := tests.MakeBlock(remoteBc, block2, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block3)

	block4 := tests.MakeBlock(remoteBc, block3, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	remoteBc.PutBlockByCoinbase(block4)

	storage2, _ := storage.NewMemoryStorage()

	dpos2 := consensus.NewDpos()
	bc := core.NewBlockChain(dpos2, storage2)
	bc.MakeGenesisBlock(voters)
	bc.PutBlockByCoinbase(bc.GenesisBlock)
	// fmt.Printf("%v\n", bc.GenesisBlock.Hash())

	block11 := rlpEncode(block1)
	bc.PutBlockIfParentExist(block11)
	b, _ := bc.GetBlockByHash(block11.Hash())
	assert.Equal(t, block11.Hash(), b.Hash(), "")

	block44 := rlpEncode(block4)
	bc.PutBlockIfParentExist(block44)
	b, _ = bc.GetBlockByHash(block44.Hash())
	assert.Nil(t, b, "")

	block33 := rlpEncode(block3)
	bc.PutBlockIfParentExist(block33)
	b, _ = bc.GetBlockByHash(block33.Hash())
	assert.Nil(t, b, "")

	block22 := rlpEncode(block2)
	bc.PutBlockIfParentExist(block22)
	b, _ = bc.GetBlockByHash(block22.Hash())
	assert.NotNil(t, b, "")

	b, _ = bc.GetBlockByHash(block33.Hash())
	assert.NotNil(t, b, "")

	b, _ = bc.GetBlockByHash(block33.Hash())
	assert.NotNil(t, b, "")

}

// /*
//      N0  LIB
//    /   \
// N1       N2
// |        |
// N4		 N3
// |        |
// N5       N6
// |
// N7
// */
// At PutBlockByCoinbase SetTail call RebuildBlockHeight
func TestRebuildBlockHeight(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	bc := core.NewBlockChain(dpos, storage)
	bc.MakeGenesisBlock(voters)
	bc.PutBlockByCoinbase(bc.GenesisBlock)

	block1 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	bc.PutBlockByCoinbase(block1)
	b1, _ := bc.GetBlockByHash(block1.Hash())
	b2, _ := bc.GetBlockByHeight(block1.Header.Height)
	assert.Equal(t, b1.Hash(), b2.Hash(), "")
	assert.Equal(t, uint64(1), block1.Header.Height, "")

	block2 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(2), tests.None, nil)
	bc.PutBlockByCoinbase(block2)
	b1, _ = bc.GetBlockByHash(block2.Hash())
	b2, _ = bc.GetBlockByHeight(block2.Header.Height)
	assert.Equal(t, b1.Hash(), b2.Hash(), "")
	assert.Equal(t, uint64(1), block2.Header.Height, "")

	block3 := tests.MakeBlock(bc, block2, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(3), tests.None, nil)
	bc.PutBlockByCoinbase(block3)
	b1, _ = bc.GetBlockByHash(block3.Hash())
	b2, _ = bc.GetBlockByHeight(block3.Header.Height)
	assert.Equal(t, b1.Hash(), b2.Hash(), "")
	assert.Equal(t, uint64(2), block3.Header.Height, "")
	b, _ := bc.GetBlockByHeight(uint64(1))
	assert.Equal(t, block2.Hash(), b.Hash(), "")

	block4 := tests.MakeBlock(bc, block1, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(4), tests.None, nil)
	bc.PutBlockByCoinbase(block4)
	b1, _ = bc.GetBlockByHash(block4.Hash())
	b2, _ = bc.GetBlockByHeight(block4.Header.Height)
	assert.Equal(t, uint64(2), block4.Header.Height, "")
	assert.Equal(t, b1.Hash(), b2.Hash(), "")
	b, _ = bc.GetBlockByHeight(uint64(1))
	assert.Equal(t, block1.Hash(), b.Hash(), "")

	block5 := tests.MakeBlock(bc, block4, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(5), tests.None, nil)
	bc.PutBlockByCoinbase(block5)
	b1, _ = bc.GetBlockByHash(block5.Hash())
	b2, _ = bc.GetBlockByHeight(block5.Header.Height)
	assert.Equal(t, b1.Hash(), b2.Hash(), "")
	assert.Equal(t, uint64(3), block5.Header.Height, "")
	b, _ = bc.GetBlockByHeight(uint64(2))
	assert.Equal(t, block4.Hash(), b.Hash(), "")

	block6 := tests.MakeBlock(bc, block3, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(6), tests.None, nil)
	bc.PutBlockByCoinbase(block6)
	b1, _ = bc.GetBlockByHash(block6.Hash())
	b2, _ = bc.GetBlockByHeight(block6.Header.Height)
	assert.Equal(t, b1.Hash(), b2.Hash(), "")
	assert.Equal(t, uint64(3), block6.Header.Height, "")
	b, _ = bc.GetBlockByHeight(uint64(2))
	assert.Equal(t, block3.Hash(), b.Hash(), "")

	block7 := tests.MakeBlock(bc, block5, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(7), tests.None, nil)
	bc.PutBlockByCoinbase(block7)
	b1, _ = bc.GetBlockByHash(block7.Hash())
	b2, _ = bc.GetBlockByHeight(block7.Header.Height)
	assert.Equal(t, b1.Hash(), b2.Hash(), "")
	assert.Equal(t, uint64(4), block7.Header.Height, "")

	b, _ = bc.GetBlockByHeight(uint64(1))
	assert.Equal(t, block1.Hash(), b.Hash(), "")
	b, _ = bc.GetBlockByHeight(uint64(2))
	assert.Equal(t, block4.Hash(), b.Hash(), "")
	b, _ = bc.GetBlockByHeight(uint64(3))
	assert.Equal(t, block5.Hash(), b.Hash(), "")
	b, _ = bc.GetBlockByHeight(uint64(4))
	assert.Equal(t, block7.Hash(), b.Hash(), "")
}

/*
// 	N0
// 	|
// 	N1
//    /    \
// N2        N3
// |        /  \
// N6(LIB)	N4  N5
// |
// N7
// |
// N8
// */
func TestRemoveOrphanBlock(t *testing.T) {
	config := tests.MakeConfig()
	voters := tests.MakeVoterAccountsFromConfig(config)
	storage, _ := storage.NewMemoryStorage()

	dpos := consensus.NewDpos()
	bc := core.NewBlockChain(dpos, storage)
	bc.MakeGenesisBlock(voters)
	bc.PutBlockByCoinbase(bc.GenesisBlock)

	block1 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
	bc.PutBlockByCoinbase(block1)

	block2 := tests.MakeBlock(bc, block1, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(2), tests.None, nil)
	bc.PutBlockByCoinbase(block2)

	block3 := tests.MakeBlock(bc, block1, tests.Addr2, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(3), tests.None, nil)
	bc.PutBlockByCoinbase(block3)

	block4 := tests.MakeBlock(bc, block3, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(4), tests.None, nil)
	bc.PutBlockByCoinbase(block4)

	block5 := tests.MakeBlock(bc, block3, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(5), tests.None, nil)
	bc.PutBlockByCoinbase(block5)

	block6 := tests.MakeBlock(bc, block2, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(6), tests.None, nil)
	bc.PutBlockByCoinbase(block6)

	block7 := tests.MakeBlock(bc, block6, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(7), tests.None, nil)
	bc.PutBlockByCoinbase(block7)

	block8 := tests.MakeBlock(bc, block7, tests.Addr2, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(8), tests.None, nil)
	bc.PutBlockByCoinbase(block8)

	bc.Lib = block6
	bc.SetTail(block6)
	bc.RemoveOrphanBlock()
	b, err := bc.GetBlockByHash(block4.Hash())
	assert.NotNil(t, err, "")
	assert.Nil(t, b, "")

}
