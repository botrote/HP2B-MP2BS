# mp2bs 
## PUSH-PULL test

Guideline for PUSH-PULL test

<img width="80%" src="https://github.com/MCNL-HGU/mp2bs/assets/52683010/b8d122f2-eacc-467b-8821-e92d54012a22"/>

## Build

```sh
go build -o anchor anchor.go
go build -o peer0 peer0.go
go build -o peers peers.go
```

You can also use `go run **.go` directly.

## Example of Usage 

```sh
clear; ./anchor -c pushpullTest/anchor.toml -t {PEER0 IP}
clear; ./peer0 -c pushpullTest/peer0.toml -i 0
clear; ./peers -c pushpullTest/peer1.toml -i 1
clear; ./peers -c pushpullTest/peer2.toml -i 2
clear; ./peers -c pushpullTest/peer3.toml -i 3
clear; ./peers -c pushpullTest/peer4.toml -i 4
```

Execution order: {PEER0|PEER1|PEER2|PEER3|PEER4} -> ANCHOR (must be last execution)

> Note: Due to hardcoding, the index of peer0 must be 0. Also, the anchor file must be last run.

In this example, peer1 will stop sending to child node(peer3) and peer3 will switch to PULL mode.
As a reuslt, peer3 can receive block data from other nodes.