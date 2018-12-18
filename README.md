# go-simplechain
It is pilot project for blockchain

## TODO

## run
```
./demo -config ../../conf/sample1.json
./demo -config ../../conf/sample2.json
./demo -config ../../conf/sample3.json
```

## rpc
```
#accounts
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "accounts", "params":[]}' http://localhost:8080/jrpc

#getBalance
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getBalance", "params":["0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"]}' http://localhost:8080/jrpc

#getTransactionCount
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionCount", "params":["0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"]}' http://localhost:8080/jrpc

#sendTransaction
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0","to": "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1","amount": "1", "nonce": "1"}
}' http://localhost:8080/jrpc

#getTransactionByHash
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionByHash", "params":["0xb017e021b8b2deba156941f32ee2e6c53c767a13749fba2533c7a30616ff48c3"]}' http://localhost:8080/jrpc

```

## Reference
* https://github.com/nebulasio/go-nebulas 
* https://github.com/ethereum/go-ethereum