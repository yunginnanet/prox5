package pxndscvm

import (
	"sync"
	"time"
)

// Statistics is used to encapsulate various swampy stats
type Statistics struct {
	// Valid4 is the amount of SOCKS4 proxies validated
	Valid4 int
	// Valid4a is the amount of SOCKS4a proxies validated
	Valid4a int
	// Valid5 is the amount of SOCKS5 proxies validated
	Valid5 int

	// Dispensed is a simple ticker to keep track of proxies dispensed via our getters
	Dispensed int

	// Stale is the amount of proxies that failed our stale policy upon dispensing
	Stale int

	// birthday represents the time we started checking proxies with this pool
	birthday time.Time

	mu *sync.Mutex
}

func (stats *Statistics) dispense() {
	stats.Dispensed++
}

func (stats *Statistics) stale() {

	stats.Stale++
}

func (stats *Statistics) v4() {
	stats.Valid4++
}

func (stats *Statistics) v4a() {
	stats.Valid4a++
}

func (stats *Statistics) v5() {
	stats.Valid5++
}

// GetUptime returns the total lifetime duration of our pool.
func (stats *Statistics) GetUptime() time.Duration {
	return time.Since(stats.birthday)
}
