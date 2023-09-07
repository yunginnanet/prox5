package prox5

import (
	"sync/atomic"
	"time"
)

// Statistics is used to encapsulate various proxy engine stats
type Statistics struct {
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

	badAccounted       int64
	accountingLastDone time.Time
}

func (stats *Statistics) dispense() {
	atomic.AddInt64(&stats.Dispensed, 1)
}

func (stats *Statistics) stale() {
	atomic.AddInt64(&stats.Stale, 1)
}

func (stats *Statistics) v4() {
	atomic.AddInt64(&stats.Valid4, 1)
}

func (stats *Statistics) v4a() {
	atomic.AddInt64(&stats.Valid4a, 1)
}

func (stats *Statistics) v5() {
	atomic.AddInt64(&stats.Valid5, 1)
}

func (stats *Statistics) http() {
	atomic.AddInt64(&stats.ValidHTTP, 1)
}

// GetTotalValidated retrieves our grand total validated proxy count.
func (p5 *ProxyEngine) GetTotalValidated() int {
	stats := p5.GetStatistics()

	total := int64(0)
	for _, val := range []*int64{&stats.Valid4a, &stats.Valid4, &stats.Valid5, &stats.ValidHTTP} {
		atomic.AddInt64(&total, atomic.LoadInt64(val))
	}

	return int(total)
}

func (p5 *ProxyEngine) GetTotalBad() int64 {
	p5.badProx.Patrons.DeleteExpired()
	return int64(p5.badProx.Patrons.ItemCount())
}

// GetUptime returns the total lifetime duration of our pool.
func (stats *Statistics) GetUptime() time.Duration {
	return time.Since(stats.birthday)
}
