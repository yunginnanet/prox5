package scaler

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/panjf2000/ants/v2"
)

type autoScalerState uint32

// for tests
var debugSwitch = false

func init() {
	if os.Getenv("PROX5_SCALER_DEBUG") != "" {
		debugSwitch = true
	}
}

func debug(msg string) {
	if debugSwitch {
		println(msg)
	}
}

func noopMsg(validated, dispensed int64, as *AutoScaler) string {
	if !debugSwitch {
		return ""
	}
	return fmt.Sprintf("noop: validated: %d, dispensed: %d, mod: %d, max: %d, threshold: %d",
		validated, dispensed, atomic.LoadInt64(as.mod), atomic.LoadInt64(as.Max), atomic.LoadInt64(as.Threshold))
}

const (
	stateDisabled autoScalerState = iota
	stateIdle
	stateScalingUp
	stateScalingDown
)

type AutoScaler struct {
	Max       *int64
	state     autoScalerState
	baseline  *int64
	mod       *int64
	Threshold *int64
}

func NewAutoScaler(baseline int, max int, differenceThreshold int) *AutoScaler {
	zero64 := int64(0)
	max64 := int64(max)
	diff64 := int64(differenceThreshold)
	baseline64 := int64(baseline)
	return &AutoScaler{
		baseline:  &baseline64,
		state:     stateDisabled,
		mod:       &zero64,
		Max:       &max64,
		Threshold: &diff64,
	}
}

func (as *AutoScaler) Disable() {
	atomic.StoreUint32((*uint32)(&as.state), uint32(stateDisabled))
}

func (as *AutoScaler) Enable() {
	atomic.StoreUint32((*uint32)(&as.state), uint32(stateIdle))
}

func (as *AutoScaler) IsOn() bool {
	return !atomic.CompareAndSwapUint32((*uint32)(&as.state), uint32(stateDisabled), uint32(stateDisabled))
}

func (as *AutoScaler) StateString() string {
	switch autoScalerState(atomic.LoadUint32((*uint32)(&as.state))) {
	case stateDisabled:
		return "disabled"
	case stateIdle:
		return "idle"
	case stateScalingUp:
		return "scaling up"
	case stateScalingDown:
		return "scaling down"
	default:
		return "unknown"
	}
}

func (as *AutoScaler) SetMax(max int) {
	atomic.StoreInt64(as.Max, int64(max))
}

func (as *AutoScaler) SetThreshold(threshold int) {
	atomic.StoreInt64(as.Threshold, int64(threshold))
}

func (as *AutoScaler) SetBaseline(baseline int) {
	atomic.StoreInt64(as.baseline, int64(baseline))
}

// ScaleAnts scales the pool, it returns true if the pool scale has been changed, and false if not.
func (as *AutoScaler) ScaleAnts(pool *ants.Pool, validated int64, dispensed int64) bool {
	if dispensed > validated {
		// consider panicing here...
		debug("dispensed > validated (FUBAR)")
		dispensed = validated
	}
	if atomic.LoadInt64(as.mod) < 0 {
		panic("scaler.go: scaler mod is negative")
	}
	if !as.IsOn() {
		debug("AutoScaler is off")
		// try to get us back to baseline if the scaler is disabled but we're not there yet.
		switch {
		case atomic.LoadInt64(as.mod) == 0:
			debug("off and not dirty")
			return false
		case atomic.LoadInt64(as.mod) > 0:
			debug("off: mod > 0")
			// we're dirty, but the scaler is off, so we need to get back to baseline
			if !(pool.Cap() > int(atomic.LoadInt64(as.baseline))) {
				debug("off: mod > 0, but pool is at baseline...")
				return false
			} else {
				debug("off and dirty: pool cap > baseline, scaling down")
				pool.Tune(pool.Cap() - 1)
				atomic.AddInt64(as.mod, -1)
				return true
			}
		default:
			debug("off: default (no-op)")
		}
		return false
	}

	sPtr := (*uint32)(&as.state)

	idle := atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateIdle))

	needScaleUp := (validated-dispensed < atomic.LoadInt64(as.Threshold)) &&
		(atomic.LoadInt64(as.mod) < atomic.LoadInt64(as.Max))

	needScaleDown := atomic.LoadInt64(as.mod) > 0 &&
		((validated - dispensed) > atomic.LoadInt64(as.Threshold))

	noop := ((idle && !needScaleUp && !needScaleDown) ||
		(needScaleUp && atomic.LoadInt64(as.mod) >= atomic.LoadInt64(as.Max)) ||
		(validated < atomic.LoadInt64(as.Threshold))) && atomic.LoadInt64(as.mod) == 0

	switch {
	case noop:
		debug(noopMsg(validated, dispensed, as))
		return false
	case ((!needScaleUp && !needScaleDown) || atomic.LoadInt64(as.mod) == 0) && !idle:
		debug("not scaling up or down or mod is 0, and not idle, setting idle")
		atomic.StoreUint32(sPtr, uint32(stateIdle))
		return false
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateScalingUp)):
		debug("scaling up")
		atomic.AddInt64(as.mod, 1)
		pool.Tune(pool.Cap() + 1)
		return true
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingUp), uint32(stateScalingUp)):
		debug("scaling up (already scaling up)")
		atomic.AddInt64(as.mod, 1)
		pool.Tune(pool.Cap() + 1)
		return true
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingDown), uint32(stateScalingUp)):
		debug("scaling up (was scaling down)")
		atomic.AddInt64(as.mod, 1)
		pool.Tune(pool.Cap() + 1)
		return true
	case needScaleDown && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingUp), uint32(stateScalingDown)):
		debug("scaling down (was scaling up)")
		atomic.AddInt64(as.mod, -1)
		pool.Tune(pool.Cap() - 1)
		return true
	case needScaleDown && atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateScalingDown)):
		debug("scaling down (was idle)")
		atomic.AddInt64(as.mod, -1)
		pool.Tune(pool.Cap() - 1)
		return true
	case needScaleDown && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingDown), uint32(stateScalingDown)):
		debug("scaling down (already scaling down)")
		atomic.AddInt64(as.mod, -1)
		pool.Tune(pool.Cap() - 1)
		return true
	default:
		debug("default (no-op)")
		return false
	}
}
