# ens-lite
Resolve ENS names without downloading full Ethereum blockchain. The resolver syncs the blockchain headers and verifies names against
the state tree.

Examples (make sure go-ethereum is installed first):
```
cd $GOPATH/src/github.com/cpacia/ens-lite/cmd/ens-lite
go install
ens-lite start
ens-lite resolve somename.eth
```

Or:
```
curl http://localhost:31313/somename.eth
```

Or as a library:
```go
client, _ = ens.NewENSLiteClient("/path/to/datadir")
go client.start()
client.Resolve("somename.eth")
```
