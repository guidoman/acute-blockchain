package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestComputeHashForBlock(t *testing.T) {
	block := getTestBlock()
	s := fmt.Sprintf("%d%d%s%s%s", block.Index, block.Timestamp, block.PreviousHash, block.Data, block.Hash)
	h := sha256.New()
	h.Write([]byte(s))
	expected := hex.EncodeToString(h.Sum(nil))
	got := computeHashForBlock(block)
	if expected != got {
		t.Errorf("computeHashForBlock() = %v, want %v", got, expected)
	}
}

func TestGenerateNextBlock(t *testing.T) {
	block := generateNextBlock("some block Data")
	if block.Index != genesisBlock.Index+1 {
		t.Errorf("generateNextBlock() Index = %v, want %v", block.Index, genesisBlock.Index+1)
	}
	if block.PreviousHash != genesisBlock.Hash {
		t.Errorf("generateNextBlock() PreviousHash = %v, want %v", block.PreviousHash, genesisBlock.Hash)
	}
}

func TestGetLatestBlock(t *testing.T) {
	block := getLatestBlock()
	if block != genesisBlock {
		t.Errorf("getLatestBlock() first block not equal to genesis block")
	}
}

func TestGetCurrentTimestamp(t *testing.T) {
	now := time.Now().UnixNano()
	tolerance := 1000.0
	ts := getCurrentTimestamp()
	diff := math.Abs(float64(ts) - float64(now))
	if diff > tolerance {
		t.Errorf("getCurrentTimestamp() = %v, want %v +/- %v", ts, now, tolerance)
	}
}

func getTestBlock() Block {
	return Block{0, getCurrentTimestamp(), "prev", "Data", ""}
}

func TestGetBlocksEndpoint(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/blocks", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getBlocksEndpoint)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	j, _ := json.Marshal(blockchain)
	expected := strings.TrimRight(string(j), "\n")
	got := strings.TrimRight(rr.Body.String(), "\n")
	if got != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			got, expected)
	}

}
