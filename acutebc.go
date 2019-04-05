package acutebc

import (
	"encoding/hex"
	"crypto/sha256"
	// "fmt"
)


// Block is the main block data type
type Block struct {
	index int64
	timestamp int64
	previousHash string
	data string
	hash string
}

// ComputeHash computes the hash of a block
// func ComputeHash(block Block) string {
func ComputeHash() string {
	// return sha256.Sum256([]byte("hello world"))
	s := "hello world"
    h := sha256.New()
    h.Write([]byte(s))
	hash := hex.EncodeToString(h.Sum(nil))
	return hash
}
