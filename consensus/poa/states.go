package poa

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/trie"
	"github.com/sirupsen/logrus"
)

type PoaState struct {
	Snapshot  *trie.Trie
	Voter     *trie.Trie
	Signer    *trie.Trie
	firstVote bool
}

func newTrie(rootHash []byte, storage storage.Storage, needChangelog bool) *trie.Trie {
	tr, err := trie.NewTrie(rootHash, storage, false)
	if err != nil {
		// if err == trie.ErrNotFound {
		// 	return tr, nil
		// }
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
		return nil
	}
	return tr
}

/* Make new state by rootHash and initialized by blockNumber*/
func NewInitState(rootHash common.Hash, blockNumber uint64, storage storage.Storage) (state *PoaState, err error) {
	var rootHashByte []byte
	if rootHash == (common.Hash{}) {
		rootHashByte = nil
	} else {
		rootHashByte = rootHash[:]
	}

	tr := newTrie(rootHashByte, storage, false)
	state = new(PoaState)
	state.Snapshot = tr
	if rootHashByte == nil {
		state.Signer = newTrie(nil, storage, false)
		state.Voter = newTrie(nil, storage, false)
		state.firstVote = true
		return state, err
	} else {
		signersHash, votersHash, err := state.Get(blockNumber)
		state.Signer = newTrie(signersHash[:], storage, false)
		if votersHash == (common.Hash{}) {
			state.Voter = newTrie(nil, storage, false)
		} else {
			state.Voter = newTrie(votersHash[:], storage, false)
		}
		state.firstVote = true
		return state, err
	}
}

func (cs *PoaState) Put(blockNumber uint64) error {
	vals := make([]byte, 0)
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return err
	}
	vals = append(vals, cs.Signer.RootHash()...)
	vals = append(vals, cs.Voter.RootHash()...)
	_, err = cs.Snapshot.Put(crypto.Sha3b256(keyEncodedBytes), vals)
	if err != nil {
		return err
	}
	return nil
}

func (cs *PoaState) Get(blockNumber uint64) (common.Hash, common.Hash, error) {
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	//TODO: check minimum key size
	encbytes, err := cs.Snapshot.Get(crypto.Sha3b256(keyEncodedBytes))
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	if len(encbytes) < common.HashLength {
		return common.Hash{}, common.Hash{}, errors.New("Bytes lenght must be more than 32 bits")
	}
	//if cs.Voter' size is 0, cs.Voter.RootHash() is 0
	if len(encbytes) == common.HashLength {
		return common.BytesToHash(encbytes[:common.HashLength]),
			common.Hash{},
			nil
	} else {
		return common.BytesToHash(encbytes[:common.HashLength]),
			common.BytesToHash(encbytes[common.HashLength:]),
			nil
	}

}

func (cs *PoaState) ValidVote(address common.Address, join bool) bool {
	_, err := cs.Signer.Get(address[:])
	if err != nil {
		return join
	}
	return !join
}

func (cs *PoaState) Vote(signer, candidate common.Address, join bool) bool {
	// Ensure the vote is meaningful
	if !cs.ValidVote(candidate, join) {
		return false
	}
	cs.Voter.Put(append(signer[:], candidate[:]...), []byte{})
	return true
}

func (cs *PoaState) signers() (addresses []common.Address, err error) {
	iter, err := cs.Signer.Iterator(nil)
	if err != nil {
		return nil, err
	}
	addresses = make([]common.Address, 0)
	exist, _ := iter.Next()
	for exist {
		addresses = append(addresses, common.BytesToAddress(iter.Key()))
		exist, err = iter.Next()
	}
	return addresses, nil
}

func (cs *PoaState) RefreshSigner() (err error) {
	targetAddress := common.Address{}
	candidate := make(map[common.Address]int)
	_signers, err := cs.signers()
	if err != nil {
		return err
	}

	iter, err := cs.Voter.Iterator(nil)
	if err != nil {
		return err
	}

	exist, err := iter.Next()
	if err != nil {
		return err
	}
	for exist {
		c := common.BytesToAddress(iter.Key()[common.AddressLength:])

		_, v := candidate[c]
		if v {
			candidate[c] += 1
		} else {
			candidate[c] = 1
		}

		if candidate[c] > len(_signers)/2 {
			_, err := cs.Signer.Get(c[:])
			if err != nil {
				if err == trie.ErrNotFound {
					_, err = cs.Signer.Put(c[:], []byte{})
					if err != nil {
						return err
					}
				} else {
					log.CLog().WithFields(logrus.Fields{}).Panic(err)
				}
			} else {
				cs.Signer.Del(c[:])
			}
			targetAddress = c
			break
		}
		exist, err = iter.Next()
	}
	if len(targetAddress) > 0 {
		iter, _ := cs.Voter.Iterator(nil)
		exist, _ := iter.Next()
		for exist {
			k := iter.Key()
			if common.BytesToAddress(k[:common.AddressLength]) == targetAddress || common.BytesToAddress(k[common.AddressLength:]) == targetAddress {
				cs.Voter.Del(iter.Key())
			}
			exist, err = iter.Next()
		}
	}
	return nil
}

type signersAscending []common.Address

func (s signersAscending) Len() int           { return len(s) }
func (s signersAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s signersAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (cs *PoaState) GetMiners() (signers []common.Address, err error) {
	signers = []common.Address{}
	iter, err := cs.Signer.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, err := iter.Next()
	if err != nil {
		return nil, err
	}
	for exist {
		k := iter.Key()
		signers = append(signers, common.BytesToAddress(k))
		exist, _ = iter.Next()
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(signersAscending(signers))
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
