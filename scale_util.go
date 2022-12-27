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
		totalChanFullNow := atomic.LoadInt64(&p5.stats.timesChannelFull)
		accountedFor := atomic.LoadInt64(&p5.stats.badAccounted) + atomic.LoadInt64(&p5.stats.fullChanAccounted)
		netFactors := (totalBadNow + totalChanFullNow) - accountedFor
		if time.Since(p5.stats.accountingLastDone) > 400*time.Millisecond && netFactors > 0 {
			bad = int64(netFactors)
			if p5.DebugEnabled() {
				p5.DebugLogger.Printf("accounting: %d bad, %d full, %d accounted, %d net factors",
					totalBadNow, totalChanFullNow, accountedFor, netFactors)
			}
			atomic.AddInt64(&p5.stats.badAccounted, 1)
			atomic.AddInt64(&p5.stats.fullChanAccounted, 1)
			p5.stats.accountingLastDone = time.Now()
		}
		// this shouldn't happen..?
		if bad < 0 {
			panic("scale_util.go: bad < 0")
		}
		if p5.pool.IsClosed() {
			return
		}

		totalValidated := int64(p5.GetTotalValidated())
		totalConsidered := int64(atomic.LoadInt64(&p5.GetStatistics().Dispensed)) + bad

		// if we are considering more than we have validated, cap it at validated so that it registers properly.
		// additionally, signal the dialer to slow down a little.
		if totalConsidered > totalValidated {
			sleep = true
			totalConsidered = totalValidated
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
