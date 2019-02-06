package core_test

import (
	"math/big"
	"testing"

	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/consensus"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/tests"
	"github.com/stretchr/testify/assert"
)

func TestGenesisBlock(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage, _ := storage.NewMemoryStorage()

	cs := consensus.NewDpos(nil)
	bc := core.NewBlockChain(storage)
	bc.Setup(cs, voters)
	bc.MakeGenesisBlock(voters)

	assert.Equal(t, voters[0].Address, bc.GenesisBlock.Header.Coinbase, "")
	assert.Equal(t, bc.GenesisBlock.Header.SnapshotVoterTime, uint64(0), "")

	//check order by balance
	minerGroup, _, _ := bc.GenesisBlock.MinerState.GetMinerGroup(bc, bc.GenesisBlock)
	assert.Equal(t, voters[0].Address, minerGroup[0], "")
	assert.Equal(t, voters[2].Address, minerGroup[1], "")
	assert.Equal(t, voters[1].Address, minerGroup[2], "")
}

func TestLoadBlockChainFromStorage(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage01, _ := storage.NewMemoryStorage()
	storage02, _ := storage.NewMemoryStorage()
	storage1, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewDpos(nil), consensus.NewPoa(nil, storage01)} {
		// cs := consensus.NewDpos()
		remoteBc := core.NewBlockChain(storage1)
		remoteBc.Setup(cs, voters)
		remoteBc.MakeGenesisBlock(voters)
		remoteBc.PutBlockByCoinbase(remoteBc.GenesisBlock)

		var cs2 core.Consensus
		if cs.ConsensusType() == "DPOS" {
			cs2 = consensus.NewDpos(nil)
		} else {
			cs2 = consensus.NewPoa(nil, storage02)
		}
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs2, voters)
		bc.LoadBlockChainFromStorage()

		assert.Equal(t, remoteBc.GenesisBlock.Hash(), bc.GenesisBlock.Hash(), "")
		assert.Equal(t, remoteBc.GenesisBlock.AccountState.RootHash(), bc.GenesisBlock.AccountState.RootHash(), "")
		assert.Equal(t, remoteBc.GenesisBlock.TransactionState.RootHash(), bc.GenesisBlock.TransactionState.RootHash(), "")
		if cs.ConsensusType() == "DPOS" {
			assert.Equal(t, remoteBc.GenesisBlock.VoterState.RootHash(), bc.GenesisBlock.VoterState.RootHash(), "")
		}
	}
}

func TestSetup(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()
	storage01, _ := storage.NewMemoryStorage()
	storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewDpos(nil), consensus.NewPoa(nil, storage01)} {
		remoteBc := core.NewBlockChain(storage1)
		remoteBc.Setup(cs, voters)

		var cs2 core.Consensus
		if cs.ConsensusType() == "DPOS" {
			cs2 = consensus.NewDpos(nil)
		} else {
			cs2 = consensus.NewPoa(nil, storage02)
			cs2.(*consensus.Poa).Period = 3
		}
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs2, voters)
		bc.LoadBlockChainFromStorage()

		assert.Equal(t, remoteBc.GenesisBlock.Hash(), bc.GenesisBlock.Hash(), "")
		assert.Equal(t, remoteBc.GenesisBlock.AccountState.RootHash(), bc.GenesisBlock.AccountState.RootHash(), "")
		assert.Equal(t, remoteBc.GenesisBlock.TransactionState.RootHash(), bc.GenesisBlock.TransactionState.RootHash(), "")
		if cs.ConsensusType() == "DPOS" {
			assert.Equal(t, remoteBc.GenesisBlock.VoterState.RootHash(), bc.GenesisBlock.VoterState.RootHash(), "")
		}
	}
}

func TestStorage(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()
	storage01, _ := storage.NewMemoryStorage()
	// storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewDpos(nil), consensus.NewPoa(nil, storage01)} {
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs, voters)
		bc.MakeGenesisBlock(voters)

		bc.PutBlockByCoinbase(bc.GenesisBlock)

		b1 := bc.GetBlockByHeight(0)
		assert.Equal(t, uint64(0), b1.Header.Height, "")
		assert.Equal(t, bc.GenesisBlock.Hash(), b1.Hash(), "")

		b2 := bc.GetBlockByHash(bc.GenesisBlock.Hash())
		assert.Equal(t, uint64(0), b2.Header.Height, "")
		assert.Equal(t, bc.GenesisBlock.Hash(), b2.Hash(), "")

		b3 := bc.GetBlockByHash(common.Hash{0x01})
		assert.Nil(t, b3, "")

		h := core.Header{}
		h.ParentHash = b1.Hash()
		block := core.Block{BaseBlock: core.BaseBlock{Header: &h}}
		trueFase := bc.HasParentInBlockChain(&block)
		assert.Equal(t, true, trueFase, "")
		h.ParentHash.SetBytes([]byte{0x01})
		trueFase = bc.HasParentInBlockChain(&block)
		assert.Equal(t, false, trueFase, "")
	}
}

type MockNode struct {
}
func (node *MockNode) HandleStream(s libnet.Stream) {

}
func (node *MockNode) SendMessage(message *net.Message, peerID peer.ID) {

}
func (node *MockNode) SendMessageToRandomNode(message *net.Message) {

}
func (node *MockNode) BroadcastMessage(message *net.Message) {}

type MockBlockChainService struct {
	bc *core.BlockChain
}

func NewMockBlockChainService(bc *core.BlockChain) *MockBlockChainService {
	bcs := MockBlockChainService{
		bc: bc,
	}
	return &bcs
}
func (bcs *MockBlockChainService) Start() {
	go func() {
		for {
			select {
			case <-bcs.bc.MessageToRandomNode:
			case <-bcs.bc.NewTXMessage:
			}
		}
	}()
}

func TestMakeBlockChain(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)

	storage01, _ := storage.NewMemoryStorage()
	storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		storage1, _ := storage.NewMemoryStorage()
		remoteBc := core.NewBlockChain(storage1)
		remoteBc.Setup(cs, voters)

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

		var cs2 core.Consensus
		if cs.ConsensusType() == "DPOS" {
			cs2 = consensus.NewDpos(nil)
		} else {
			cs2 = consensus.NewPoa(nil, storage02)
		}
		bc := core.NewBlockChain(storage2)
		if cs.ConsensusType() == "POA" {
			cs2.(*consensus.Poa).Period = 3
		}
		bc.Setup(cs2, voters)
		bcs := NewMockBlockChainService(bc)
		bcs.Start()

		bc.PutBlockIfParentExist(block1)
		b := bc.GetBlockByHash(block1.Hash())
		assert.Equal(t, block1.Hash(), b.Hash(), "")

		bc.PutBlockIfParentExist(block4)
		b = bc.GetBlockByHash(block4.Hash())
		assert.Nil(t, b, "")

		bc.PutBlockIfParentExist(block3)
		b = bc.GetBlockByHash(block3.Hash())
		assert.Nil(t, b, "")

		bc.PutBlockIfParentExist(block2)
		b = bc.GetBlockByHash(block2.Hash())
		assert.NotNil(t, b, "")

		b = bc.GetBlockByHash(block3.Hash())
		assert.NotNil(t, b, "")

		b = bc.GetBlockByHash(block4.Hash())
		assert.NotNil(t, b, "")
	}
}

func rlpEncode(block *core.Block) *core.Block {
	message, _ := net.NewRLPMessage(net.MsgNewBlock, block)
	block2 := core.Block{}
	rlp.DecodeBytes(message.Payload, &block2)
	return &block2
}

func TestMakeBlockChainWhenRlpEncode(t *testing.T) {
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()
	storage01, _ := storage.NewMemoryStorage()
	storage02, _ := storage.NewMemoryStorage()

	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		remoteBc := core.NewBlockChain(storage1)
		remoteBc.Setup(cs, voters)

		block1 := tests.MakeBlock(remoteBc, remoteBc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		remoteBc.PutBlockByCoinbase(block1)

		block2 := tests.MakeBlock(remoteBc, block1, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		remoteBc.PutBlockByCoinbase(block2)

		block3 := tests.MakeBlock(remoteBc, block2, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		remoteBc.PutBlockByCoinbase(block3)

		block4 := tests.MakeBlock(remoteBc, block3, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		remoteBc.PutBlockByCoinbase(block4)

		storage2, _ := storage.NewMemoryStorage()

		var cs2 core.Consensus
		if cs.ConsensusType() == "DPOS" {
			cs2 = consensus.NewDpos(nil)
		} else {
			cs2 = consensus.NewPoa(nil, storage02)
			cs2.(*consensus.Poa).Period = 3
		}
		bc := core.NewBlockChain(storage2)
		bc.Setup(cs2, voters)
		bcs := NewMockBlockChainService(bc)
		bcs.Start()

		block11 := rlpEncode(block1)
		bc.PutBlockIfParentExist(block11)
		b := bc.GetBlockByHash(block11.Hash())
		assert.Equal(t, block11.Hash(), b.Hash(), "")

		block44 := rlpEncode(block4)
		bc.PutBlockIfParentExist(block44)
		b = bc.GetBlockByHash(block44.Hash())
		assert.Nil(t, b, "")

		block33 := rlpEncode(block3)
		bc.PutBlockIfParentExist(block33)
		b = bc.GetBlockByHash(block33.Hash())
		assert.Nil(t, b, "")

		block22 := rlpEncode(block2)
		bc.PutBlockIfParentExist(block22)
		b = bc.GetBlockByHash(block22.Hash())
		assert.NotNil(t, b, "")

		b = bc.GetBlockByHash(block33.Hash())
		assert.NotNil(t, b, "")

		b = bc.GetBlockByHash(block33.Hash())
		assert.NotNil(t, b, "")
	}
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
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()
	storage01, _ := storage.NewMemoryStorage()
	// storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs, voters)
		bc.MakeGenesisBlock(voters)
		bc.PutBlockByCoinbase(bc.GenesisBlock)

		block1 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block1)
		b1 := bc.GetBlockByHash(block1.Hash())
		b2 := bc.GetBlockByHeight(block1.Header.Height)
		assert.Equal(t, b1.Hash(), b2.Hash(), "")
		assert.Equal(t, uint64(1), block1.Header.Height, "")

		block2 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(2), tests.None, nil)
		bc.PutBlockByCoinbase(block2)
		b1 = bc.GetBlockByHash(block2.Hash())
		b2 = bc.GetBlockByHeight(block2.Header.Height)
		assert.Equal(t, b1.Hash(), b2.Hash(), "")
		assert.Equal(t, uint64(1), block2.Header.Height, "")

		block3 := tests.MakeBlock(bc, block2, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(3), tests.None, nil)
		bc.PutBlockByCoinbase(block3)
		b1 = bc.GetBlockByHash(block3.Hash())
		b2 = bc.GetBlockByHeight(block3.Header.Height)
		assert.Equal(t, b1.Hash(), b2.Hash(), "")
		assert.Equal(t, uint64(2), block3.Header.Height, "")
		b := bc.GetBlockByHeight(uint64(1))
		assert.Equal(t, block2.Hash(), b.Hash(), "")

		block4 := tests.MakeBlock(bc, block1, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(4), tests.None, nil)
		bc.PutBlockByCoinbase(block4)
		b1 = bc.GetBlockByHash(block4.Hash())
		b2 = bc.GetBlockByHeight(block4.Header.Height)
		assert.Equal(t, uint64(2), block4.Header.Height, "")
		assert.Equal(t, b1.Hash(), b2.Hash(), "")
		b = bc.GetBlockByHeight(uint64(1))
		assert.Equal(t, block1.Hash(), b.Hash(), "")

		block5 := tests.MakeBlock(bc, block4, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(5), tests.None, nil)
		bc.PutBlockByCoinbase(block5)
		b1 = bc.GetBlockByHash(block5.Hash())
		b2 = bc.GetBlockByHeight(block5.Header.Height)
		assert.Equal(t, b1.Hash(), b2.Hash(), "")
		assert.Equal(t, uint64(3), block5.Header.Height, "")
		b = bc.GetBlockByHeight(uint64(2))
		assert.Equal(t, block4.Hash(), b.Hash(), "")

		block6 := tests.MakeBlock(bc, block3, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(6), tests.None, nil)
		bc.PutBlockByCoinbase(block6)
		b1 = bc.GetBlockByHash(block6.Hash())
		b2 = bc.GetBlockByHeight(block6.Header.Height)
		assert.Equal(t, b1.Hash(), b2.Hash(), "")
		assert.Equal(t, uint64(3), block6.Header.Height, "")
		b = bc.GetBlockByHeight(uint64(2))
		assert.Equal(t, block3.Hash(), b.Hash(), "")

		block7 := tests.MakeBlock(bc, block5, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(7), tests.None, nil)
		bc.PutBlockByCoinbase(block7)
		b1 = bc.GetBlockByHash(block7.Hash())
		b2 = bc.GetBlockByHeight(block7.Header.Height)
		assert.Equal(t, b1.Hash(), b2.Hash(), "")
		assert.Equal(t, uint64(4), block7.Header.Height, "")

		b = bc.GetBlockByHeight(uint64(1))
		assert.Equal(t, block1.Hash(), b.Hash(), "")
		b = bc.GetBlockByHeight(uint64(2))
		assert.Equal(t, block4.Hash(), b.Hash(), "")
		b = bc.GetBlockByHeight(uint64(3))
		assert.Equal(t, block5.Hash(), b.Hash(), "")
		b = bc.GetBlockByHeight(uint64(4))
		assert.Equal(t, block7.Hash(), b.Hash(), "")
	}
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
	voters := cmd.MakeVoterAccountsFromConfig(config)
	storage1, _ := storage.NewMemoryStorage()
	storage01, _ := storage.NewMemoryStorage()
	// storage02, _ := storage.NewMemoryStorage()
	for _, cs := range []core.Consensus{consensus.NewPoa(nil, storage01), consensus.NewDpos(nil)} {
		bc := core.NewBlockChain(storage1)
		bc.Setup(cs, voters)
		// bc.MakeGenesisBlock(voters)
		// bc.PutBlockByCoinbase(bc.GenesisBlock)

		block1 := tests.MakeBlock(bc, bc.GenesisBlock, tests.Addr0, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(1), tests.None, nil)
		bc.PutBlockByCoinbase(block1)

		block2 := tests.MakeBlock(bc, block1, tests.Addr1, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(2), tests.None, nil)
		bc.PutBlockByCoinbase(block2)

		//2,3 block same tx hash
		block3 := tests.MakeBlock(bc, block1, tests.Addr2, tests.Addr0, tests.Addr1, new(big.Int).SetUint64(2), tests.None, nil)
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

		bc.SetLib(block6)
		bc.SetTail(block6)

		assert.Equal(t, bc.TxPool.Len(), 0, "")
		bc.RemoveOrphanBlock()
		b := bc.GetBlockByHash(block3.Hash())
		assert.Nil(t, b, "")

		b = bc.GetBlockByHash(block4.Hash())
		assert.Nil(t, b, "")

		b = bc.GetBlockByHash(block5.Hash())
		assert.Nil(t, b, "")
		// N3 same tx,  N4,N5 different tx
		assert.Equal(t, bc.TxPool.Len(), 2, "")
	}
}
