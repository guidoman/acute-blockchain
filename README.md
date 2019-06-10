[![Build Status](https://travis-ci.org/guidoman/acute-blockchain.png)](https://travis-ci.org/guidoman/acute-blockchain)

# Acute Blockchain - A minimalistic blockchain implementation in Go

This is a minimalistic implementation of a blockchain with the Go programming language (Golang). It is basically a port of [Naivechain](https://github.com/lhartikk/naivechain).

## Purpose
* Understand the Naivechain concept better
* Learn Go (in particular Websockets)
* Compare Javascript and Go implementations

## Quick start

To get started:
1. Clone or fork this repository
2. Install **go** version 1.11 or later
3. Run `go get`

Then, you can start two or more nodes running on different posts (inside two separate terminals):

```
PORT=3000 go run main.go hub.go
PORT=3001 go run main.go hub.go
```

You can then add the second process as peer of the first one:

```
curl -XPOST -d 'localhost:3001' http://localhost:3000/addPeer
```

You can then mine a block on a given node by posting some data to the `mineBlock` API:

```
curl -XPOST -d 'Second block mined by first process' http://localhost:3000/mineBlock
```

You can check that the blockchain on the two nodes is identical by running the following commands:

```
curl http://localhost:3000/blocks
curl http://localhost:3001/blocks
```

The output of both commands should be something like the following (pretty-printed):

```
[
  {
    "index": 0,
    "timestamp": 0,
    "previousHas": "1465154705",
    "data": "בראשית",
    "hash": "816534932c2b7154836da6afc367695e6337db8a921823784c14378abed4f7d7"
  },
  {
    "index": 1,
    "timestamp": 1560182697635975000,
    "previousHas": "816534932c2b7154836da6afc367695e6337db8a921823784c14378abed4f7d7",
    "data": "Second block mined by first process",
    "hash": "2723d64bab200e7bf23f10a602153c50ddb8bcba1770d56b20f4ef00db3842e2"
  }
]
```