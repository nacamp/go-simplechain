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

	//test getMinerSize
	minerSize, _ := cs.getMinerSize(block)
	assert.Equal(t, 3, minerSize)
}

/*
//----------    Consensus  ----------------//
func (cs *Poa) UpdateLIB() {
	bc := cs.bc
	block := bc.Tail
	//FIXME: consider timestamp
	miners := make(map[common.Address]bool)
	turn := 1
	if block.Header.Height == 0 {
		return
	}
	firstMinerSize, err := cs.getMinerSize(block)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{
			"Msg": err,
		}).Warning("getMinerSize")
		return
	}
	if firstMinerSize < 3 {
		log.CLog().WithFields(logrus.Fields{
			"Size": firstMinerSize,
		}).Debug("At least 3 node are needed")
		return
	}
	for bc.Lib.Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		size, err := cs.getMinerSize(block)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Warning("getMinerSize")
			return
		}
		if firstMinerSize != size {
			return
		}
		if turn == firstMinerSize {
			if len(miners) == firstMinerSize*2/3+1 {
				bc.SetLib(block)
				log.CLog().WithFields(logrus.Fields{
					"Height": block.Header.Height,
				}).Info("Updated Lib")
				return
			}
			miners = make(map[common.Address]bool)
			miners[block.Header.Coinbase] = true
			turn = 0
		}
		block = bc.GetBlockByHash(block.Header.ParentHash)
		turn++
	}
	return
}

func (cs *Poa) ConsensusType() string {
	return "POA"
}




func (cs *Poa) Verify(block *core.Block) error {
	// parentBlock := cs.bc.GetBlockByHash(block.Header.ParentHash)
	// if parentBlock == nil {
	// 	return errors.New("parent block is nil")
	// }
	// miners, err := cs.GetMiners(parentBlock.Hash())
	// if err != nil {
	// 	return err
	// }

	state := block.ConsensusState.(*PoaState)
	miners, err := state.GetMiners()
	if err != nil {
		return err
	}
	index := (block.Header.Time % (uint64(len(miners)) * cs.Period)) / cs.Period
	if miners[index] != block.Header.Coinbase {
		return errors.New("This turn is not this miner's turn ")
	}
	return nil
}



func (cs *Poa) SaveState(block *core.Block) (err error) {
	state := block.ConsensusState.(*PoaState)
	state.RefreshSigner()
	// accs := block.AccountState

	// electedTime := state.GetNewElectedTime(state.ElectedTime, block.Header.Time, 3, 3, 3)

	// if electedTime == block.Header.Time {
	// 	miners, err := state.GetNewRoundMiners(block.Header.Time, 3)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	state.MinersHash, err = state.PutMiners(miners)
	// 	state.ElectedTime = block.Header.Time

	// 	iter, err := state.Voter.Iterator(nil)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	exist, _ := iter.Next()
	// 	for exist {
	// 		account := accs.GetAccount(common.BytesToAddress(iter.Key()))
	// 		account.CalcSetTotalPeggedStake()
	// 		accs.PutAccount(account)
	// 		exist, err = iter.Next()
	// 	}
	// 	//reset voter if this round is new
	// 	state.Voter, err = trie.NewTrie(nil, cs.bc.Storage, false)
	// }
	err = state.Put(block.Header.Height)
	if err != nil {
		return err
	}
	return nil

	// if err := cs.VerifyMinerTurn(block); err != nil {
	// 	return err
	// }
	// snapshot, err := cs.LoadSnapshot(block.Header.ParentHash)
	// if err != nil {
	// 	log.CLog().Warning(err)
	// 	return err
	// }
	// if snapshot == nil {
	// 	return errors.New("Snapshot is nil")
	// }
	// newSnap := snapshot.Copy()
	// newSnap.BlockHash = block.Hash()
	// for _, tx := range block.Transactions {

	// 	if tx.Payload != nil {
	// 		//TODO: fix after dpos coding
	// 		// authorize := bool(true)
	// 		// rlp.DecodeBytes(tx.Payload, &authorize)
	// 		// if newSnap.Cast(tx.From, tx.To, authorize) {
	// 		// 	newSnap.Apply()
	// 		// }
	// 		break
	// 	}
	// }
	// h := newSnap.CalcHash()
	// if h != block.Header.SnapshotHash {
	// 	return errors.New("Hash is different")
	// }
	// newSnap.Store(cs.Storage)
	return nil
}



func (cs *Poa) LoadState(block *core.Block) (state core.ConsensusState, err error) {
	bc := cs.bc

	state, err = NewInitState(block.Header.ConsensusHash, block.Header.Height, bc.Storage)
	if err != nil {
		return nil, err
	}
	return state, nil
}

*/
