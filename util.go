package Prox5

import (
	quiccmaffs "math/rand"
	"time"
)

const (
	grn = "\033[32m"
	red = "\033[31m"
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
	quiccmaffs.Seed(time.Now().UnixNano())
	return quiccmaffs.Uint32()
}

func randSleep() {
	quiccmaffs.Seed(time.Now().UnixNano())
	time.Sleep(time.Duration(quiccmaffs.Intn(500)) * time.Millisecond)
}

func allNil(obj ...interface{}) bool {
	for _, o := range obj {
		if o != nil {
			return false
		}
	}
	return true
}
