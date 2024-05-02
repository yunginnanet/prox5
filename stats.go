package prox5

import (
	"sync/atomic"
	"time"
)

// Statistics is used to encapsulate various proxy engine stats
type Statistics struct {
	// Valid4 is the amount of SOCKS4 proxies validated
	Valid4 *atomic.Int64
	// Valid4a is the amount of SOCKS4a proxies validated
	Valid4a *atomic.Int64
	// Valid5 is the amount of SOCKS5 proxies validated
	Valid5 *atomic.Int64
	// ValidHTTP is the amount of HTTP proxies validated
	ValidHTTP *atomic.Int64
	// Dispensed is a simple ticker to keep track of proxies dispensed via our getters
	Dispensed *atomic.Int64
	// Stale is the amount of proxies that failed our stale policy upon dispensing
	Stale *atomic.Int64
	// Checked is the amount of proxies we've checked.
	Checked *atomic.Int64
	// birthday represents the time we started checking proxies with this pool
	birthday *atomic.Pointer[time.Time]

	badAccounted       *atomic.Int64
	accountingLastDone *atomic.Pointer[time.Time]
}

func (stats *Statistics) dispense() {
	stats.Dispensed.Add(1)
}

func (stats *Statistics) stale() {
	stats.Stale.Add(1)
}

func (stats *Statistics) v4() {
	stats.Valid4.Add(1)
}

func (stats *Statistics) v4a() {
	stats.Valid4a.Add(1)
}

func (stats *Statistics) v5() {
	stats.Valid5.Add(1)
}

func (stats *Statistics) http() {
	stats.ValidHTTP.Add(1)
}

// GetTotalValidated retrieves our grand total validated proxy count.
func (p5 *ProxyEngine) GetTotalValidated() int64 {
	stats := p5.GetStatistics()

	total := int64(0)
	for _, val := range []*atomic.Int64{stats.Valid4a, stats.Valid4, stats.Valid5, stats.ValidHTTP} {
		total += val.Load()
	}
	return total
}

func (p5 *ProxyEngine) GetTotalBad() int64 {
	p5.badProx.Patrons.DeleteExpired()
	return int64(p5.badProx.Patrons.ItemCount())
}

// GetUptime returns the total lifetime duration of our pool.
func (stats *Statistics) GetUptime() time.Duration {
	return time.Since(*stats.birthday.Load())
}
