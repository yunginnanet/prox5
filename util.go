package pxndscvm

import (
	"crypto/rand"
	"encoding/binary"
)

const (
	grn = "\033[32m"
	red =  "\033[31m"
	ylw = "\033[33m"
	rst = "\033[0m"
)

// randStrChoice returns a random element from the given string slice
func randStrChoice(choices []string) string {
	strlen := len(choices)
	n := uint32(0)
	if strlen > 0 {
		n = getRandomUint32() % uint32(strlen)
	}
	return choices[n]
}

// getRandomUint32 retrieves a cryptographically sound random 32 bit unsigned little endian integer
func getRandomUint32() uint32 {
	b := make([]byte, 8192)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(b)
}
