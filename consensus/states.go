package consensus

import (
	"bytes"
	"math/big"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/trie"
)

type Miner struct {
	// Timestamp  uint64
	Address    common.Address
	nonce      uint64
	MinerGroup []common.Address
	VoterHash  common.Hash
}

type MinerState struct {
	Trie *trie.Trie
}

func (ms *MinerState) RootHash() (hash common.Hash) {
	copy(hash[:], ms.Trie.RootHash())
	return hash
}

//TODO: error
func (ms *MinerState) Put(miner *Miner) (hash common.Hash) {
	encodedBytes, _ := rlp.EncodeToBytes(miner)
	ms.Trie.Put(miner.Address[:], encodedBytes)
	copy(hash[:], ms.Trie.RootHash())
	return hash
}

//TODO: error
func (ms *MinerState) Get(address common.Address) (miner *Miner) {
	decodedBytes, err := ms.Trie.Get(address[:])
	//FIXME: TOBE
	// if err != nil && err != storage.ErrKeyNotFound {
	// 	return nil, err
	// }
	if err == nil {
		rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
		return miner
	} else {
		return &Miner{}
	}

}

//------------------------------------
type VoterState struct {
	Trie *trie.Trie
}

func (vs *VoterState) RootHash() (hash common.Hash) {
	copy(hash[:], vs.Trie.RootHash())
	return hash
}

//TODO: error
func (vs *VoterState) Put(account *core.Account) (hash common.Hash) {
	encodedBytes, _ := rlp.EncodeToBytes(account)
	vs.Trie.Put(account.Address[:], encodedBytes)
	copy(hash[:], vs.Trie.RootHash())
	return hash
}

//TODO: error
func (vs *VoterState) Get(address common.Address) (account *core.Account) {
	decodedBytes, err := vs.Trie.Get(address[:])
	//FIXME: TOBE
	// if err != nil && err != storage.ErrKeyNotFound {
	// 	return nil, err
	// }
	if err == nil {
		rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&account)
		return account
	} else {
		return &core.Account{Address: address, Balance: new(big.Int).SetUint64(0)}
	}

}
