# go-simplechain
It is pilot project for blockchain

## TODO

## run
```
./demo -config ../../conf/sample1.json
./demo -config ../../conf/sample2.json
./demo -config ../../conf/sample3.json
```

## config
```
When sample1 run,  node_pub.id is generated in node_key_path
Replace nodeid to node_pub.id
in sample2.json, sample3.json
"seeds" :["/ip4/127.0.0.1/tcp/9991/ipfs/nodeid"]
```

## account command
```
import privatekey 
./demo account import -config ../../conf/sample1.json

new address
./demo account new -config ../../conf/sample1.json
```



## sample address/privatekey
```
address :    0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d
privatekey : 0x8a21cd44e684dd2d8d9205b0bfb69339435c7bd016ebc21fddaddffd0d47ed63
address :    0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44
privatekey : 0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff
address :    0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2
privatekey : 0x47661aa6cccada84454842404ec0cca83760254191232f1d4cc11653d397ac2e
```

## rpc
```
#accounts
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "accounts", "params":[]}' http://localhost:8080/jrpc

#getBalance
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getBalance", "params":["0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d"]}' http://localhost:8080/jrpc

#getTransactionCount
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionCount", "params":["0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d"]}' http://localhost:8080/jrpc

#sendTransaction
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0x3068c6c17a079f67b3f29a9844cbf6137a2bd7a3a58f0d0eac11b8afcd4564b8e4173af7","to": "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1","amount": "1", "nonce": "1"}}' http://localhost:8080/jrpc

#sendTransaction vote evicting  when consensus is poa
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d","to": "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2","amount": "0", "nonce": "1", "payload":"false"}
}' http://localhost:8080/jrpc

#getTransactionByHash
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionByHash", "params":["0x3e551a9b75dcb741b8b4a2bc431c8c21ce65bbf37889365dbff874d4351bde89"]}' http://localhost:8080/jrpc

#newAccount
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "newAccount", "params":["password"]}' http://localhost:8080/jrpc

#unlock
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "unlock", "params": {"address": "0x3068c6c17a079f67b3f29a9844cbf6137a2bd7a3a58f0d0eac11b8afcd4564b8e4173af7","password": "password","timeout": 300}}' http://localhost:8080/jrpc
```

## Reference
* https://github.com/nebulasio/go-nebulas 
* https://github.com/ethereum/go-ethereum