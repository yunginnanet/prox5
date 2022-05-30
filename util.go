package prox5

import (
	"git.tcp.direct/kayos/common/entropy"
)

const (
	grn = "\033[32m"
	red = "\033[31m"
	ylw = "\033[33m"
	rst = "\033[0m"
)

// randStrChoice returns a random element from the given string slice.
func randStrChoice(choices []string) string {
	return entropy.RandomStrChoice(choices)
}

func randSleep() {
	entropy.RandSleepMS(200)
}
