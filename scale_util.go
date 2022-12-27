package prox5

import (
	"strconv"
	"time"
)

var scaleTimer = time.NewTicker(100 * time.Millisecond)

func (p5 *ProxyEngine) scaleDbg() {
	if !p5.DebugEnabled() {
		return
	}
	msg := strs.Get()
	msg.MustWriteString("job spawner auto scaling, new count: ")
	msg.MustWriteString(strconv.Itoa(p5.pool.Cap()))
	p5.dbgPrint(msg)
}

func (p5 *ProxyEngine) scale() {
	select {
	case <-scaleTimer.C:
		if p5.pool.IsClosed() {
			return
		}
		if p5.scaler.ScaleAnts(p5.pool, p5.GetTotalValidated(), p5.GetStatistics().Dispensed) {
			p5.scaleDbg()
		}
	default:
		return
	}
}
