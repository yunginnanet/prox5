package prox5

import (
	"strconv"
	"sync/atomic"
	"time"
)

func (p5 *ProxyEngine) scaleDbg() {
	if !p5.DebugEnabled() {
		return
	}
	msg := strs.Get()
	msg.MustWriteString("job spawner auto scaling, new count: ")
	msg.MustWriteString(strconv.Itoa(p5.pool.Cap()))
	p5.dbgPrint(msg)
}

func (p5 *ProxyEngine) scale() (sleep bool) {
	select {
	case <-p5.scaleTimer.C:
		bad := int64(0)
		totalBadNow := p5.GetTotalBad()
		accountedFor := p5.stats.badAccounted.Load()
		netFactors := totalBadNow - accountedFor
		if time.Since(*p5.stats.accountingLastDone.Load()) > 5*time.Second && netFactors > 0 {
			bad = netFactors
			if p5.DebugEnabled() {
				p5.DebugLogger.Printf("accounting: %d bad - %d accounted for = %d net factors",
					totalBadNow, accountedFor, netFactors)
			}
			tnow := time.Now()
			p5.stats.accountingLastDone.Store(&tnow)
		}
		// this shouldn't happen..?
		if bad < 0 {
			panic("scale_util.go: bad < 0")
		}
		if p5.pool.IsClosed() {
			return
		}

		totalValidated := p5.GetTotalValidated()
		totalConsidered := p5.GetStatistics().Dispensed.Load() + bad

		// if we are considering more than we have validated, cap it at validated so that it registers properly.
		// additionally, signal the dialer to slow down a little.
		if totalConsidered >= totalValidated {
			sleep = true
			totalConsidered = totalValidated - atomic.LoadInt64(p5.scaler.Threshold)/2
		}

		if p5.scaler.ScaleAnts(
			p5.pool,
			totalValidated,
			totalConsidered,
		) {
			p5.scaleDbg()
		}
	default:
		return
	}
	return
}
