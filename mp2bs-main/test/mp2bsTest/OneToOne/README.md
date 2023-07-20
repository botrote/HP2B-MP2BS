# mp2bs 
## one-to-one test

Guideline for one-to-one test

<img width="80%" src="https://github.com/MCNL-HGU/mp2bs/assets/52683010/5e566b93-3b3e-4a36-bc94-a70d11478f33"/>

## Build

```sh
go build -o anchor anchor.go
go build -o peer0 peer0.go
go build -o peers peers.go
```

You can also use `go run **.go` directly.

## Example of Usage 

```sh
clear; ./anchor -c OneToOne/anchor.toml -t {PEER0 IP}
clear; ./peer0 -c OneToOne/peer0.toml -i 0
clear; ./peers -c OneToOne/peers.toml -i 2
```

Execution order: {PEER0|PEER#} -> ANCHOR (must be last execution)

> Note: Due to hardcoding, the index of peer0 must be 0, and the index of peers must be greater than 1.

If you want to trigger traffic for a specific second..

```sh
clear; ./peer0 -c OneToOne/peer0.toml -i 0 -t true -r 100 -e 10 
```

> Note: The execution may not last as expected because the data is generated from the values you enter.

-r means the throughput estimated by the user and -e means the total running time desired by the user.
In this example, rate is 100mbit and execution time is 10sec. 