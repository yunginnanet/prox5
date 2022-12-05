package scaler

import (
	"sync/atomic"

	"github.com/panjf2000/ants/v2"
)

type autoScalerState uint32

// for tests
var debugSwitch = false

func debug(msg string) {
	//goland:noinspection ALL
	if debugSwitch {
		println(msg)
	}
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
	Threshold *int64
}

func NewAutoScaler(max int, differenceThreshold int) *AutoScaler {
	zero64 := int64(0)
	max64 := int64(max)
	diff64 := int64(differenceThreshold)
	return &AutoScaler{
		baseline:  &zero64,
		state:     stateDisabled,
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
func (as *AutoScaler) ScaleAnts(pool *ants.Pool, validated int, dispensed int64) bool {
	if !as.IsOn() {
		debug("AutoScaler is off")
		// try to get us back to baseline if the scaler is disabled but we're not there yet.
		switch {
		case atomic.LoadInt64(as.baseline) == 0:
			debug("off: baseline is 0")
			// baseline is already 0, nothing to do
		case atomic.LoadInt64(as.baseline) == int64(pool.Cap()):
			debug("off: baseline is pool cap")
			// baseline is already at the pool's capacity, nothing to do, store the new baseline
			atomic.StoreInt64(as.baseline, 0)
		case pool.Cap() > int(atomic.LoadInt64(as.baseline)):
			debug("off: pool cap > baseline")
			pool.Tune(pool.Cap() - 1)
			return true
		case pool.Cap() < int(atomic.LoadInt64(as.baseline)):
			debug("off: pool cap < baseline")
			pool.Tune(pool.Cap() + 1)
			return true
		default:
			debug("off: default (no-op)")
			// no-op
		}
		return false
	}

	sPtr := (*uint32)(&as.state)

	fresh := atomic.LoadInt64(as.baseline) == 0

	idle := atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateIdle))

	needScaleUp := (validated-int(dispensed) < int(atomic.LoadInt64(as.Threshold))) &&
		(pool.Cap() < int(atomic.LoadInt64(as.Max)))

	needScaleDown := !fresh &&
		((validated - int(dispensed)) > int(atomic.LoadInt64(as.Threshold))) &&
		(int64(pool.Cap()) > atomic.LoadInt64(as.baseline))

	noop := (idle && !needScaleUp && !needScaleDown) ||
		(needScaleUp && pool.Cap() >= int(atomic.LoadInt64(as.Max))) ||
		(validated < int(atomic.LoadInt64(as.Threshold)))

	switch {
	case noop:
		debug("noop")
		return false
	case !needScaleUp && !needScaleDown && !idle:
		debug("not scaling up or down and not idle")
		atomic.StoreUint32(sPtr, uint32(stateIdle))
		return false
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateScalingUp)):
		debug("scaling up")
		atomic.StoreInt64(as.baseline, int64(pool.Cap()))
		pool.Tune(pool.Cap() + 1)
		return true
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingUp), uint32(stateScalingUp)):
		debug("scaling up (already scaling up)")
		pool.Tune(pool.Cap() + 1)
		return true
	case needScaleDown && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingUp), uint32(stateScalingDown)):
		debug("scaling down (was scaling up)")
		pool.Tune(pool.Cap() - 1)
		return true
	case needScaleDown && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingDown), uint32(stateScalingDown)):
		debug("scaling down (already scaling down)")
		pool.Tune(pool.Cap() - 1)
		return true
	default:
		debug("default (no-op)")
		return false
	}
}
