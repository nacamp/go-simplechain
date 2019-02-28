package poa

import (
	"fmt"
	"testing"

	"github.com/nacamp/go-simplechain/tests"
	"github.com/nacamp/go-simplechain/trie"

	"github.com/stretchr/testify/assert"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/storage"
)

func signers(state *PoaState) []common.Address {
	addresses, _ := state.signers()
	return addresses
}

func voters(state *PoaState) [][]byte {
	doubleAddress := make([][]byte, 0)
	iter, err := state.Voter.Iterator(nil)
	if err != nil {
		return doubleAddress
	}
	exist, _ := iter.Next()
	for exist {
		doubleAddress = append(doubleAddress, iter.Key())
		exist, _ = iter.Next()
	}
	return doubleAddress
}

func TestState(t *testing.T) {
	_storage, _ := storage.NewMemoryStorage()
	state, err := NewInitState(common.Hash{}, 0, _storage)
	assert.NoError(t, err)
	_ = state

	state.Signer.Put(common.FromHex(tests.Addr0), []byte{})
	state.Signer.Put(common.FromHex(tests.Addr1), []byte{})
	state.Signer.Put(common.FromHex(tests.Addr2), []byte{})

	//test signers
	addresses, _ := state.signers()
	assert.Equal(t, common.FromHex(tests.Addr0), addresses[0][:])
	assert.Equal(t, common.FromHex(tests.Addr1), addresses[1][:])
	assert.Equal(t, common.FromHex(tests.Addr2), addresses[2][:])

	//test ValidVote
	var newAddr = "0x1df75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2"
	assert.False(t, state.ValidVote(common.HexToAddress(newAddr), false))
	assert.False(t, state.ValidVote(common.HexToAddress(tests.Addr0), true))
	assert.True(t, state.ValidVote(common.HexToAddress(newAddr), true))
	assert.True(t, state.ValidVote(common.HexToAddress(tests.Addr0), false))

	//test Vote & RefreshSigner
	//FIXME: ban self vote, check if voter is signer
	assert.False(t, state.Vote(common.HexToAddress(tests.Addr0), common.HexToAddress(newAddr), false))

	assert.True(t, state.Vote(common.HexToAddress(tests.Addr0), common.HexToAddress(newAddr), true))
	assert.Equal(t, 1, len(voters(state)))
	_ = state.RefreshSigner()
	assert.Equal(t, 3, len(signers(state)))

	assert.True(t, state.Vote(common.HexToAddress(tests.Addr1), common.HexToAddress(newAddr), true))
	assert.Equal(t, 2, len(voters(state)))
	_ = state.RefreshSigner()
	assert.Equal(t, 4, len(signers(state)))
	assert.Equal(t, 0, len(voters(state)))

	//test put & get
	err = state.Put(1)
	fmt.Println(err)
	signersHash, votersHash, err := state.Get(1)
	//if voters size is 0, return common.Hash{}
	assert.Equal(t, common.Hash{}, votersHash)
	//if votersHash is common.Hash{}, use nil
	tr, _ := trie.NewTrie(nil, _storage, false)
	assert.Equal(t, state.Voter.RootHash(), tr.RootHash())
	tr, _ = trie.NewTrie(signersHash[:], _storage, false)
	assert.Equal(t, state.Signer.RootHash(), tr.RootHash())

	//test NewInitState
	state2, err := NewInitState(state.RootHash(), 1, _storage)
	assert.NoError(t, err)
	assert.Equal(t, state.Signer.RootHash(), state2.Signer.RootHash())

}

/*
	// 	//TODO: who voter?
	for _, v := range voters {
		state.Vote(v.Address, v.Address, true)
		state.RefreshSigner()
	}
	state.Put(block.Header.Height)
func NewInitState(rootHash common.Hash, blockNumber uint64, storage storage.Storage) (state *PoaState, err error) {
	var rootHashByte []byte
	if rootHash == (common.Hash{}) {
		rootHashByte = nil
	} else {
		rootHashByte = rootHash[:]
	}

	tr, err := trie.NewTrie(rootHashByte, storage, false)
	if err != nil {
		return nil, err
	}

	state = new(PoaState)
	state.Snapshot = tr
	signersHash, votersHash, err := state.Get(blockNumber)
	if err != nil {
		if err == trie.ErrNotFound {
			tr2, err := trie.NewTrie(nil, storage, false)
			state.Signer = tr2
			tr3, err := trie.NewTrie(nil, storage, false)
			state.Voter = tr3
			return state, err
		}
		return nil, err
	}

	tr2, err := trie.NewTrie(signersHash[:], storage, false)
	state.Signer = tr2
	tr3, err := trie.NewTrie(votersHash[:], storage, false)
	state.Voter = tr3
	state.firstVote = true
	return state, err
}


func (s *PoaState) CalcHash() (hash common.Hash) {
	blob, _ := json.Marshal(s)
	hasher := sha3.New256()
	hasher.Write(blob)
	hasher.Sum(hash[:0])
	return hash
}


func (cs *PoaState) GetMiners() (signers []common.Address, err error) {
	signers = []common.Address{}
	iter, err := cs.Signer.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, _ := iter.Next()
	for exist {
		k := iter.Key()
		signers = append(signers, common.BytesToAddress(k))
		exist, err = iter.Next()
	}
	return signers, nil
}

func (cs *PoaState) Clone() (core.ConsensusState, error) {
	tr1, err1 := cs.Voter.Clone()
	if err1 != nil {
		return nil, err1
	}
	tr2, err2 := cs.Signer.Clone()
	if err2 != nil {
		return nil, err2
	}
	tr3, err3 := cs.Snapshot.Clone()
	if err3 != nil {
		return nil, err3
	}
	return &PoaState{
		Voter:     tr1,
		Signer:    tr2,
		Snapshot:  tr3,
		firstVote: true,
	}, nil
}

func (cs *PoaState) ExecuteTransaction(block *core.Block, txIndex int, account *core.Account) (err error) {
	tx := block.Transactions[txIndex]
	if tx.From == block.Header.Coinbase && cs.firstVote {
		cs.firstVote = false
	} else {
		return errors.New("This tx is not validated")
	}
	if tx.Payload.Code == core.TxCVoteStake {
		cs.Vote(tx.From, tx.To, true)
	} else if tx.Payload.Code == core.TxCVoteUnStake {
		cs.Vote(tx.From, tx.To, false)
	}
	return nil
}

func (cs *PoaState) RootHash() (hash common.Hash) {
	copy(hash[:], cs.Snapshot.RootHash())
	return hash
}
*/
