# ens-lite
Resolve ENS names into DNS records or Ethereum addresses without downloading full Ethereum blockchain. The resolver syncs the blockchain headers and verifies names against
the state tree.

Examples (make sure go-ethereum is installed first):
```
cd $GOPATH/src/github.com/cpacia/ens-lite/cmd/ens-lite
go install
ens-lite start

ens-lite resolve somename.eth
ens-lite lookup somename.eth
ens-lite address somename.eth
```

Or:
```
curl http://localhost:31313/resolver/dns/somename.eth
curl http://localhost:31313/resolver/dns/somename.eth?lookup=true
curl http://localhost:31313/resolver/address/somename.eth
```

Or as a library:
```go
client, _ = ens.NewENSLiteClient("/path/to/datadir")
go client.start()
client.ResolveDNS("somename.eth")
client.ResolveAddress("somename.eth")
```
