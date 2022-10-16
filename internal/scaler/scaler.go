package scaler

import (
	"sync/atomic"

	"github.com/panjf2000/ants/v2"
)

type autoScalerState uint32

const (
	stateDisabled autoScalerState = iota
	stateIdle
	stateScalingUp
	stateScalingDown
)

type AutoScaler struct {
	old       *int64
	Max       int
	state     autoScalerState
	Threshold int
}

func NewAutoScaler(max int, differenceThreshold int) *AutoScaler {
	zero := int64(0)
	return &AutoScaler{
		old:       &zero,
		state:     stateDisabled,
		Max:       max,
		Threshold: differenceThreshold,
	}
}

func (as *AutoScaler) Disable() {
	atomic.StoreUint32((*uint32)(&as.state), uint32(stateDisabled))
}

func (as *AutoScaler) Enable() {
	atomic.StoreUint32((*uint32)(&as.state), uint32(stateIdle))
}

func (as *AutoScaler) IsOn() bool {
	return as.state != stateDisabled
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

// TODO: test cases

func (as *AutoScaler) ScaleAnts(pool *ants.Pool, validated int, dispensed int64) bool {
	if atomic.CompareAndSwapUint32((*uint32)(&as.state), uint32(stateDisabled), uint32(stateDisabled)) {
		switch {
		case atomic.LoadInt64(as.old) == 0:
			// no-op
		case atomic.LoadInt64(as.old) == int64(pool.Cap()):
			atomic.StoreInt64(as.old, 0)
		case pool.Cap() > int(atomic.LoadInt64(as.old)):
			pool.Tune(pool.Cap() - 1)
			return true
		case pool.Cap() < int(atomic.LoadInt64(as.old)):
			pool.Tune(pool.Cap() + 1)
			return true
		default:
			// no-op
		}
		return false
	}

	sPtr := (*uint32)(&as.state)

	fresh := atomic.LoadInt64(as.old) == 0

	idle := atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateIdle))

	needScaleUp := (validated-int(dispensed) < as.Threshold) && (pool.Cap() < as.Max)

	needScaleDown := !fresh &&
		((validated - int(dispensed)) > as.Threshold) &&
		(int64(pool.Cap()) > atomic.LoadInt64(as.old))

	noop := (idle && !needScaleUp && !needScaleDown) || (needScaleUp && pool.Cap() >= as.Max) || (validated < as.Threshold)

	switch {
	case noop:
		return false
	case !needScaleUp && !needScaleDown && !idle:
		atomic.StoreUint32(sPtr, uint32(stateIdle))
		return false
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateScalingUp)):
		atomic.StoreInt64(as.old, int64(pool.Cap()))
		pool.Tune(pool.Cap() + 1)
		return true
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingUp), uint32(stateScalingUp)):
		pool.Tune(pool.Cap() + 1)
		return true
	case needScaleDown && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingUp), uint32(stateScalingDown)):
		pool.Tune(pool.Cap() - 1)
		return true
	case needScaleDown && atomic.CompareAndSwapUint32(sPtr, uint32(stateScalingDown), uint32(stateScalingDown)):
		pool.Tune(pool.Cap() - 1)
		return true
	default:
		return false
	}
}
