package tests

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"sort"

	"github.com/btcsuite/btcd/btcec"
	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/crypto"
)

var AddressHex0 = string("0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9")
var AddressHex1 = string("0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5")
var AddressHex2 = string("0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d")
var AddressHex3 = string("0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44")
var AddressHex4 = string("0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df")
var AddressHex5 = string("0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2")

var Keystore = map[string]string{
	AddressHex0: "0x9a84c5fb3c9136419f6408f5ccf065ef86e1b0cd09f0494c8c0aefc261b37536",
	AddressHex1: "0xe17d66fb8ce7cabd6601a8b04d08fe14fed4720782a72524458bafba587ee1fe",
	AddressHex2: "0x8a21cd44e684dd2d8d9205b0bfb69339435c7bd016ebc21fddaddffd0d47ed63",
	AddressHex3: "0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff",
	AddressHex4: "0x99cd17b1a9a303c5e6ac43d33e3fc1f86d3790f8e25564a1ad531da77a9482bc",
	AddressHex5: "0x47661aa6cccada84454842404ec0cca83760254191232f1d4cc11653d397ac2e",
}

// var AddressHex0 = string("0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d")
// var AddressHex1 = string("0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44")
// var AddressHex2 = string("0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2")

// var AddressHex3 = string("0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9")
// var AddressHex4 = string("0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5")
// var AddressHex5 = string("0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df")

// var Keystore = map[string]string{ //0, 2, 1
// 	AddressHex0: "0x8a21cd44e684dd2d8d9205b0bfb69339435c7bd016ebc21fddaddffd0d47ed63",
// 	AddressHex1: "0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff",
// 	AddressHex2: "0x47661aa6cccada84454842404ec0cca83760254191232f1d4cc11653d397ac2e",
// 	AddressHex3: "0x9a84c5fb3c9136419f6408f5ccf065ef86e1b0cd09f0494c8c0aefc261b37536",
// 	AddressHex4: "0xe17d66fb8ce7cabd6601a8b04d08fe14fed4720782a72524458bafba587ee1fe",
// 	AddressHex5: "0x99cd17b1a9a303c5e6ac43d33e3fc1f86d3790f8e25564a1ad531da77a9482bc",
// }

var Address0 = common.HexToAddress(AddressHex0)
var Address1 = common.HexToAddress(AddressHex1)
var Address2 = common.HexToAddress(AddressHex2)
var Address3 = common.HexToAddress(AddressHex3)
var Address4 = common.HexToAddress(AddressHex4)
var Address5 = common.HexToAddress(AddressHex5)

type trick int

const (
	None trick = iota
	CHANGE_COINBASE
	LangObjC
)

func MakeConfig() *cmd.Config {
	saveWallet(AddressHex0, "password", "/var/tmp/simple/keystore1.dat")

	configStr := `
	{
		"port"  : 9991,
		"rpc_address" :"127.0.0.1:8080",
		"enable_mining": true,
		"db_path" : "/var/tmp/simple/simplechain",
		"node_key_path" : "/var/tmp/simple/nodekey1",
		"keystore_file" : "/var/tmp/simple/keystore1.dat",
		"miner_address" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"miner_passphrase" : "password",
		"coinbase" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"mining_reward" : 10,
		"seeds" :  [""],
		"consensus" : 
        {
        "name":"dpos", 
        "period":3, 
        "round":3, 
		"total_miners":3,
		"difficulty"  :1000
        },
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
	}
	`
	contents := []byte(configStr)
	config := &cmd.Config{}
	json.Unmarshal([]byte(contents), config)
	return config
}

func NewConfig(turn int) *cmd.Config {
	saveWallet(AddressHex0, "password", "/var/tmp/simple/keystore1.dat")
	saveWallet(AddressHex1, "password", "/var/tmp/simple/keystore2.dat")
	saveWallet(AddressHex2, "password", "/var/tmp/simple/keystore3.dat")
	saveWallet(AddressHex3, "password", "/var/tmp/simple/keystore4.dat")
	saveWallet(AddressHex4, "password", "/var/tmp/simple/keystore5.dat")
	saveWallet(AddressHex5, "password", "/var/tmp/simple/keystore6.dat")
	configStr := []string{
		`
	{
		"port"  : 9991,
		"rpc_address" :"127.0.0.1:8080",
		"enable_mining": true,
		"db_path" : "/var/tmp/simple/simplechain",
		"node_key_path" : "/var/tmp/simple/nodekey1",
		"keystore_file" : "/var/tmp/simple/keystore1.dat",
		"miner_address" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"miner_passphrase" : "password",
		"coinbase" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"mining_reward" : 10,
		"seeds" :  [""],
		"consensus" : 
        {
        "name":"dpos", 
        "period":3, 
        "round":3, 
		"total_miners":3,
		"difficulty"  :1000
        },
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
	}
	`,
		`
	{
		"port"  : 9992,
		"rpc_address" :"127.0.0.1:8082",
		"enable_mining": true,
		"db_path" : "/var/tmp/simple/simplechain2",
		"node_key_path" : "/var/tmp/simple/nodekey2",
		"keystore_file" : "/var/tmp/simple/keystore2.dat",
		"miner_address" : "0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5",
		"miner_passphrase" : "password",
		"coinbase" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"mining_reward" : 10,
		"seeds" :  ["/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"],
		"consensus" : 
        {
        "name":"dpos", 
        "period":3, 
        "round":3, 
		"total_miners":3,
		"difficulty"  :1000
        },
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
	}
	`,
		`
	{
		"port"  : 9993,
		"rpc_address" :"127.0.0.1:8083",
		"enable_mining": true,
		"db_path" : "/var/tmp/simple/simplechain3",
		"node_key_path" : "/var/tmp/simple/nodekey3",
		"keystore_file" : "/var/tmp/simple/keystore3.dat",
		"miner_address" : "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d",
		"miner_passphrase" : "password",
		"coinbase" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"mining_reward" : 10,
		"seeds" :  ["/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"],
		"consensus" : 
        {
        "name":"dpos", 
        "period":3, 
        "round":3, 
		"total_miners":3,
		"difficulty"  :1000
        },
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
	}
	`,
		`
	{
		"port"  : 9994,
		"rpc_address" :"127.0.0.1:8084",
		"enable_mining": true,
		"db_path" : "/var/tmp/simple/simplechain4",
		"node_key_path" : "/var/tmp/simple/nodekey4",
		"keystore_file" : "/var/tmp/simple/keystore4.dat",
		"miner_address" : "0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44",
		"miner_passphrase" : "password",
		"coinbase" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"mining_reward" : 10,
		"seeds" :  ["/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"],
		"consensus" : 
        {
        "name":"dpos", 
        "period":3, 
        "round":3, 
		"total_miners":3,
		"difficulty"  :1000
        },
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
	}
	`,
		`
	{
		"port"  : 9995,
		"rpc_address" :"127.0.0.1:8085",
		"enable_mining": true,
		"db_path" : "/var/tmp/simple/simplechain5",
		"node_key_path" : "/var/tmp/simple/nodekey5",
		"keystore_file" : "/var/tmp/simple/keystore4.da5",
		"miner_address" : "0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df",
		"miner_passphrase" : "password",
		"coinbase" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"mining_reward" : 10,
		"seeds" :  ["/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"],
		"consensus" : 
        {
        "name":"dpos", 
        "period":3, 
        "round":3, 
		"total_miners":3,
		"difficulty"  :1000
        },
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
	}
	`,
		`
	{
		"port"  : 9996,
		"rpc_address" :"127.0.0.1:8086",
		"enable_mining": true,
		"db_path" : "/var/tmp/simple/simplechain6",
		"node_key_path" : "/var/tmp/simple/nodekey6",
		"keystore_file" : "/var/tmp/simple/keystore4.da6",
		"miner_address" : "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2",
		"miner_passphrase" : "password",
		"coinbase" : "0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9",
		"mining_reward" : 10,
		"seeds" :  ["/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"],
		"consensus" : 
        {
        "name":"dpos", 
        "period":3, 
        "round":3, 
		"total_miners":3,
		"difficulty"  :1000
        },
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
	}
	`,
	}
	contents := []byte(configStr[turn])
	config := &cmd.Config{}
	json.Unmarshal([]byte(contents), config)
	return config
}

// func NewConfig(turn int) *cmd.Config {
// 	saveWallet(AddressHex0, "password", "/var/tmp/simple/keystore1.dat")
// 	saveWallet(AddressHex1, "password", "/var/tmp/simple/keystore2.dat")
// 	saveWallet(AddressHex2, "password", "/var/tmp/simple/keystore3.dat")
// 	configStr := []string{
// 		`
// 	{
// 		"port"  : 9991,
// 		"rpc_address" :"127.0.0.1:8080",
// 		"enable_mining": true,
// 		"db_path" : "/var/tmp/simple/simplechain",
// 		"node_key_path" : "/var/tmp/simple/nodekey1",
// 		"keystore_file" : "/var/tmp/simple/keystore1.dat",
// 		"miner_address" : "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d",
// 		"miner_passphrase" : "password",
// 		"coinbase" : "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d",
// 		"mining_reward" : 10,
// 		"seeds" :  [""],
// 		"consensus" :
//         {
//         "name":"dpos",
//         "period":3,
//         "round":3,
// 		"total_miners":3,
// 		"difficulty"  :1000
//         },
// 		"voters" : [{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":100 },
// 					{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":20 },
// 					{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":50 }]
// 	}
// 	`,
// 		`
// 	{
// 		"port"  : 9992,
// 		"rpc_address" :"127.0.0.1:8082",
// 		"enable_mining": true,
// 		"db_path" : "/var/tmp/simple/simplechain2",
// 		"node_key_path" : "/var/tmp/simple/nodekey2",
// 		"keystore_file" : "/var/tmp/simple/keystore2.dat",
// 		"miner_address" : "0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44",
// 		"miner_passphrase" : "password",
// 		"coinbase" : "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d",
// 		"mining_reward" : 10,
// 		"seeds" :  ["/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"],
// 		"consensus" :
//         {
//         "name":"dpos",
//         "period":3,
//         "round":3,
// 		"total_miners":3,
// 		"difficulty"  :1000
//         },
// 		"voters" : [{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":100 },
// 			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":20 },
// 			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":50 }]
// 	}
// 	`,
// 		`
// 	{
// 		"port"  : 9993,
// 		"rpc_address" :"127.0.0.1:8083",
// 		"enable_mining": true,
// 		"db_path" : "/var/tmp/simple/simplechain3",
// 		"node_key_path" : "/var/tmp/simple/nodekey3",
// 		"keystore_file" : "/var/tmp/simple/keystore3.dat",
// 		"miner_address" : "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2",
// 		"miner_passphrase" : "password",
// 		"coinbase" : "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d",
// 		"mining_reward" : 10,
// 		"seeds" :  ["/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"],
// 		"consensus" :
//         {
//         "name":"dpos",
//         "period":3,
//         "round":3,
// 		"total_miners":3,
// 		"difficulty"  :1000
//         },
// 		"voters" : [{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":100 },
// 			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":20 },
// 			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":50 }]
// 	}
// 	`,
// 	}
// 	contents := []byte(configStr[turn])
// 	config := &cmd.Config{}
// 	json.Unmarshal([]byte(contents), config)
// 	return config
// }

type signersAscending []common.Address

func (s signersAscending) Len() int           { return len(s) }
func (s signersAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s signersAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func SignerSlice(signers []common.Address) []common.Address {
	sigs := make([]common.Address, 0, len(signers))
	for _, sig := range signers {
		sigs = append(sigs, sig)
	}
	sort.Sort(signersAscending(sigs))
	return sigs
}

// func MakeBlock(bc *core.BlockChain, parentBlock *core.Block, coinbase, from, to string, amount *big.Int, trickId trick, trickValue interface{}) *core.Block {
// 	h := &core.Header{}
// 	h.ParentHash = parentBlock.Hash()
// 	h.Height = parentBlock.Header.Height + 1
// 	h.Time = parentBlock.Header.Time + 3
// 	//set time to turn coinbase mining
// 	if bc.Consensus.ConsensusType() == "DPOS" {
// 		// gMs, _ := bc.GenesisBlock.MinerState.Clone()
// 		// gMinerGroup, _ := gMs.MakeMiner(bc.GenesisBlock.VoterState, 3)
// 		// for {
// 		// 	index := (h.Time % 9) / 3
// 		// 	if gMinerGroup[index] == common.HexToAddress(coinbase) {
// 		// 		break
// 		// 	}
// 		// 	h.Time++
// 		// }
// 	} else {
// 		gMinerGroup := SignerSlice(bc.Signers)
// 		for {
// 			index := (h.Time % 9) / 3
// 			if gMinerGroup[index] == common.HexToAddress(coinbase) {
// 				break
// 			}
// 			h.Time++
// 		}
// 		// snapshot, err := dpos.snapshot(block.Header.ParentHash)
// 	}
// 	block := &core.Block{BaseBlock: core.BaseBlock{Header: h}}

// 	if bc.Consensus.ConsensusType() == "DPOS" {
// 		// //voter
// 		// block.VoterState, _ = parentBlock.VoterState.Clone()
// 		// h.VoterHash = block.VoterState.RootHash()

// 		// //miner
// 		// block.MinerState, _ = parentBlock.MinerState.Clone()
// 		// minerGroup, voterBlock, _ := block.MinerState.GetMinerGroup(bc, block)

// 		// //TODO: we need to test  when voter transaction make
// 		// if voterBlock.Header.Height == block.Header.Height {
// 		// 	block.MinerState.Put(minerGroup, block.Header.VoterHash)
// 		// 	fmt.Printf("VoterHash(put), height, time, >>>%v, %v, %v\n", block.Header.Height, block.Header.Time, block.Header.VoterHash)
// 		// } else {
// 		// 	fmt.Printf("VoterHash(   ), height, time, >>>%v, %v, %v\n", block.Header.Height, block.Header.Time, block.Header.VoterHash)
// 		// }
// 	}
// 	h.Coinbase = common.HexToAddress(coinbase)
// 	// index := block.Header.Height % 3
// 	// h.Coinbase = minerGroup[index]
// 	if trickId == CHANGE_COINBASE {
// 		h.Coinbase = common.HexToAddress((trickValue).(string))
// 	}
// 	// fmt.Printf("height,index,address : %v-%v-%v\n", block.Header.Height, index, common.Bytes2Hex(h.Coinbase[:]))

// 	//account, transaction
// 	block.AccountState, _ = parentBlock.AccountState.Clone()
// 	block.TransactionState, _ = parentBlock.TransactionState.Clone()
// 	coinbaseAccount := block.AccountState.GetAccount(h.Coinbase)
// 	coinbaseAccount.AddBalance(new(big.Int).SetUint64(100))
// 	block.AccountState.PutAccount(coinbaseAccount)

// 	accs := block.AccountState
// 	txs := block.TransactionState

// 	fromAccount := accs.GetAccount(common.HexToAddress(from))
// 	tx := MakeTransaction(from, to, amount, fromAccount.Nonce+uint64(1))
// 	// tx := MakeTransaction(from, to, new(big.Int).Div(amount, new(big.Int).SetUint64(2)), fromAccount.Nonce+uint64(1))
// 	block.TransactionState.PutTransaction(tx)
// 	block.Transactions = make([]*core.Transaction, 1)
// 	block.Transactions[0] = tx
// 	fromAccount.Nonce += uint64(1)

// 	toAccount := accs.GetAccount(tx.To)
// 	fromAccount.SubBalance(tx.Amount)
// 	toAccount.AddBalance(tx.Amount)
// 	// fromAccount.SubBalance(tx.Amount)
// 	// toAccount.AddBalance(tx.Amount)

// 	accs.PutAccount(fromAccount)
// 	accs.PutAccount(toAccount)
// 	h.AccountHash = block.AccountState.RootHash()

// 	txs.PutTransaction(tx)
// 	h.TransactionHash = block.TransactionState.RootHash()

// 	if bc.Consensus.ConsensusType() == "DPOS" {
// 		// h.MinerHash = block.MinerState.RootHash()
// 	}

// 	if bc.Consensus.ConsensusType() == "POA" {
// 		// //TODO: fix temp hash
// 		// block.Header.SnapshotHash = bc.GenesisBlock.Header.SnapshotHash
// 		// // cannot use below code for cycling reference
// 		// // 	snapshot, _ := bc.Consensus.(*consensus.Poa).Snapshot(block.Header.ParentHash)
// 	}
// 	block.MakeHash()
// 	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(Keystore[coinbase]))
// 	block.Sign((*ecdsa.PrivateKey)(priv))
// 	return block
// }

func MakeTransaction(from, to string, amount *big.Int, nonce uint64) *core.Transaction {
	tx := core.NewTransaction(common.HexToAddress(from), common.HexToAddress(to), amount, nonce)
	tx.MakeHash()
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(Keystore[from]))
	tx.Sign((*ecdsa.PrivateKey)(priv))
	return tx
}

// func MakeWallet() *account.Wallet {
// 	wallet := account.NewWallet("./test_keystore.dat")
// 	for _, priv := range keystore {
// 		key := new(account.Key)
// 		key.PrivateKey = crypto.ByteToPrivateKey(common.FromHex(priv))
// 		key.Address = crypto.CreateAddressFromPrivateKey(key.PrivateKey)
// 		wallet.StoreKey(key, "test")
// 	}
// 	return wallet
// }

func saveWallet(addr, pwd, path string) {
	key := new(account.Key)
	key.PrivateKey = crypto.ByteToPrivateKey(common.FromHex(Keystore[addr]))
	key.Address = crypto.CreateAddressFromPrivateKey(key.PrivateKey)
	w := account.NewWallet(path)
	w.StoreKey(key, pwd)
}
