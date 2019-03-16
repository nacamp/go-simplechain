package pow

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/consensus/dpos"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/tests"
)

func TestPow(t *testing.T) {
	var err error
	// var block *core.Block
	//config
	config := tests.MakeConfig()
	voters := cmd.MakeVoterAccountsFromConfig(config)
	mstrg, _ := storage.NewMemoryStorage()
	cs := dpos.NewDpos(net.NewPeerStreamPool(), config.Consensus.Period, config.Consensus.Round, config.Consensus.TotalMiners)
	wallet := account.NewWallet(config.KeystoreFile)
	wallet.Load()
	err = wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
	if err != nil {
		log.CLog().Fatal(err)
	}

	cs.SetupMining(common.HexToAddress(config.MinerAddress), wallet)
	bc := core.NewBlockChain(mstrg, common.HexToAddress(config.Coinbase), uint64(config.MiningReward))

	//test MakeGenesisBlock in Setup
	bc.Setup(cs, voters)
	fmt.Println(bc.GenesisBlock.Hash())

	bc.GenesisBlock.Header.Difficulty = new(big.Int).SetUint64(0)
	//131072
	//now := uint64(time.Now().Unix())
	//now := uint64(1552645325 + 2000)
	now := uint64(0)
	d := calcDifficulty(uint64(time.Now().Unix()), bc.GenesisBlock.Header)
	bc.GenesisBlock.Header.Difficulty = new(big.Int).SetUint64(now)

	//a, _ := new(big.Int).SetString("1880629730694143", 10)
	bc.GenesisBlock.Header.Difficulty = d
	//1552645325 현재시간
	//1551887060
	//1551887060
	//1551887198PASS
	//1551887198PASS
	//1551889197PASS
	d = calcDifficulty(now+20, bc.GenesisBlock.Header)
	fmt.Printf("%+v", d)
	target := new(big.Int).Div(two256, d)
	fmt.Println(target)
	result := makeHashNonce(common.HashToBytes(bc.GenesisBlock.Hash()), uint64(132))
	//131072883423532389192164791648750371459257913741948437809479060803100646309888
	//155189418374613392140871369851362465612571928722327058729390548942767436101610
	fmt.Println("target:", len(fmt.Sprintf("%s", target)), target)
	fmt.Println("result:", len(fmt.Sprintf("%s", new(big.Int).SetBytes(result))), new(big.Int).SetBytes(result))

	//6882193669365321860012852210660599385518202572004718670727482521792129960433454455837618835558366604466843490347685542605101369737437872414126219867966599
	//11262965605146337364849074743862288017191302439499338096036016421232216581187060668974276907186238619224084848776609792043218302884121655876564616486616742
	//6882193669365321860012852210660599385518202572004718670727482521792129960433454455837618835558366604466843490347685542605101369737437872414126219867966599
	fmt.Println(rand.Int63())
	fmt.Println(rand.Int63())
	fmt.Println(rand.Int63())
	// rand.Uint64()
	for i := 0; i < 10000000; i++ {
		result := makeHashNonce(common.HashToBytes(bc.GenesisBlock.Hash()), uint64(rand.Int63()+int64(i)))
		if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
			fmt.Println(i, "break")
			fmt.Println("target:", len(fmt.Sprintf("%s", target)), target)
			fmt.Println("result:", len(fmt.Sprintf("%s", new(big.Int).SetBytes(result))), new(big.Int).SetBytes(result))
			break
		}
	}
	/*
		func (ethash *Ethash) mine(block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {


				target := new(big.Int).Div(two256, header.Difficulty)
			if new(big.Int).SetBytes(result).Cmp(target) > 0 {
				return errInvalidPoW
			}
	*/
	//fmt.Println(hashimoto(common.HashToBytes(bc.GenesisBlock.Hash()), uint64(0)))
}
