package pxndscvm

import (
	"crypto/rand"
	"encoding/binary"
)

// RandStrChoice returns a random element from the given string slice
func RandStrChoice(choices []string) string {
	strlen := len(choices)
	n := uint32(0)
	if strlen > 0 {
		n = GetRandomUint32() % uint32(strlen)
	}
	return choices[n]
}

// GetRandomUint32 retrieves a cryptographically sound random 32 bit unsigned little endian integer
func GetRandomUint32() uint32 {
	b := make([]byte, 8192)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(b)
}

func (s *Swamp) dbgPrint(str string) {
	if s.swampopt.Debug {
		println("pxndscvm: " + str)
	}
}
