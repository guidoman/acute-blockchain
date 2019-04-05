package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestComputeHash(t *testing.T) {
	block := Block{0, time.Now().UnixNano(), "prev", "data", ""}
	s := fmt.Sprintf("%d%d%s%s%s", block.index, block.timestamp, block.previousHash, block.data, block.hash)
	h := sha256.New()
	h.Write([]byte(s))
	expected := hex.EncodeToString(h.Sum(nil))
	got := ComputeHash(block)
	if expected != got {
		t.Errorf("ComputeHash() = %v, want %v", got, expected)
	}
}
