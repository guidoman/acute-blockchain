package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Block is the main block data type
type Block struct {
	index        int64
	timestamp    int64
	previousHash string
	data         string
	hash         string
}

// ComputeHash computes the hash of a block struct
func ComputeHash(block Block) string {
	s := fmt.Sprintf("%d%d%s%s%s", block.index, block.timestamp, block.previousHash, block.data, block.hash)
	h := sha256.New()
	h.Write([]byte(s))
	hash := hex.EncodeToString(h.Sum(nil))
	return hash
}

func main() {
	fmt.Println("Hello, blockchain")
}
