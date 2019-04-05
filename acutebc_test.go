package acutebc

import (
	"testing"
	"fmt"
)

func TestComputeHash(t *testing.T) {
	const hello = "Hello, world"
	hash := ComputeHash()
	fmt.Println(hash)
}