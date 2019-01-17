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
In sample2.json, sample3.json
"seeds" :["/ip4/127.0.0.1/tcp/9991/ipfs/nodeid"]

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
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d","to": "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2","amount": "1", "nonce": "1"}
}' http://localhost:8080/jrpc

#sendTransaction vote evicting  when consensus is poa
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d","to": "0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2","amount": "0", "nonce": "1", "payload":"false"}
}' http://localhost:8080/jrpc


#getTransactionByHash
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionByHash", "params":["0x3e551a9b75dcb741b8b4a2bc431c8c21ce65bbf37889365dbff874d4351bde89"]}' http://localhost:8080/jrpc

```

## Reference
* https://github.com/nebulasio/go-nebulas 
* https://github.com/ethereum/go-ethereum