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
	old                 *int64
	max                 int
	state               autoScalerState
	differenceThreshold int
}

func NewAutoScaler(max int, differenceThreshold int) *AutoScaler {
	return &AutoScaler{
		max:                 max,
		differenceThreshold: differenceThreshold,
	}
}

func (as AutoScaler) ScaleAnts(pool *ants.Pool, validated int, dispensed int64) bool {
	if atomic.CompareAndSwapUint32((*uint32)(&as.state), uint32(stateDisabled), uint32(stateDisabled)) {
		return false
	}

	sPtr := (*uint32)(&as.state)

	old := atomic.LoadInt64(as.old)
	fresh := old == 0

	idle := atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateIdle))

	needScaleUp := (validated-int(dispensed) < as.differenceThreshold) && (pool.Cap() < as.max)

	needScaleDown := !idle && !fresh &&
		(validated-int(dispensed) > as.differenceThreshold) &&
		(int64(pool.Cap()) > atomic.LoadInt64(as.old))

	noop := (idle && !needScaleUp) || (needScaleUp && pool.Cap() >= as.max)

	switch {
	case noop:
		return false
	case !needScaleUp && !needScaleDown && !idle:
		atomic.StoreUint32(sPtr, uint32(stateIdle))
		return false
	case needScaleUp && atomic.CompareAndSwapUint32(sPtr, uint32(stateIdle), uint32(stateScalingUp)):
		atomic.StoreInt64(as.old, int64(pool.Cap()))
		pool.Tune(int(*as.old) + 1)
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
