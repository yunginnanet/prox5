package prox5

import (
	"time"
)

// statistics is used to encapsulate various swampy stats
type statistics struct {
	// Valid4 is the amount of SOCKS4 proxies validated
	Valid4 int64
	// Valid4a is the amount of SOCKS4a proxies validated
	Valid4a int64
	// Valid5 is the amount of SOCKS5 proxies validated
	Valid5 int64
	// ValidHTTP is the amount of HTTP proxies validated
	ValidHTTP int64
	// Dispensed is a simple ticker to keep track of proxies dispensed via our getters
	Dispensed int64
	// Stale is the amount of proxies that failed our stale policy upon dispensing
	Stale int64
	// Checked is the amount of proxies we've checked.
	Checked int64
	// birthday represents the time we started checking proxies with this pool
	birthday time.Time
}

func (stats *statistics) dispense() {
	stats.Dispensed++
}

func (stats *statistics) stale() {
	stats.Stale++
}

func (stats *statistics) v4() {
	stats.Valid4++
}

func (stats *statistics) v4a() {
	stats.Valid4a++
}

func (stats *statistics) v5() {
	stats.Valid5++
}

func (stats *statistics) http() {
	stats.ValidHTTP++
}

// GetTotalValidated retrieves our grand total validated proxy count.
func (pe *Swamp) GetTotalValidated() int {
	stats := pe.GetStatistics()
	return int(stats.Valid4a + stats.Valid4 + stats.Valid5 + stats.ValidHTTP)
}

// GetUptime returns the total lifetime duration of our pool.
func (stats *statistics) GetUptime() time.Duration {
	return time.Since(stats.birthday)
}
