# go-simplechain
It is pilot project for blockchain


## run
```
./simple -config ../../conf/sample1.json
./simple -config ../../conf/sample2.json
./simple -config ../../conf/sample3.json
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
./simple account import -config ../../conf/sample1.json

new address
./simple account new -config ../../conf/sample1.json
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
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d","to": "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2","amount": "1", "nonce": "2"}}' http://localhost:8080/jrpc


#sendTransaction vote when consensus is dpos
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d","to": "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2","amount": "0", "nonce": "3", "payload":{"code":"0", "data":"10"}}}' http://localhost:8080/jrpc

payload.code==1  : stake
payload.code==0  : unstake
payload.data : amount


#sendTransaction vote when consensus is poa
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d","to": "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2","amount": "0", "nonce": "2", "payload":{"code":"1"}}}' http://localhost:8080/jrpc

payload.code==1  : joinning  
payload.code==0  : evicting 



#getTransactionByHash
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionByHash", "params":["0x3e551a9b75dcb741b8b4a2bc431c8c21ce65bbf37889365dbff874d4351bde89"]}' http://localhost:8080/jrpc

#newAccount
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "newAccount", "params":["password"]}' http://localhost:8080/jrpc

#unlock
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "unlock", "params": {"address": "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d","password": "password","timeout": 300}}' http://localhost:8080/jrpc
```



## Reference
* https://github.com/nebulasio/go-nebulas 
* https://github.com/ethereum/go-ethereum



## POA
### Voting
Only Miners can vote to evict another miner or to elect a new miner and vote 1 in a block they mined
When a miner or candidate receives more than one-half (1/2) vote, he can mine or is evicted.

### Mining
Miners can mined a block in their turn and blocks mined in order of over 1/2  become Last Irreversible Block