package scaler

import (
	"testing"

	"github.com/panjf2000/ants/v2"
	"github.com/ysmood/gotrace"
)

func checkLeak(t *testing.T) {
	gotrace.CheckLeak(t, 0)
}

var dummyPool *ants.Pool

func init() {
	var err error
	dummyPool, err = ants.NewPool(5, ants.WithNonblocking(false))
	if err != nil {
		panic(err)
	}
}

// I know this is a lot of lines unnecessarily. For a test, I don't care.
// With that being said, this test could be much more exhaustive.
// For now, this can serve as a sanity check.
func TestNewAutoScaler(t *testing.T) {
	checkLeak(t)
	// debugSwitch = true
	as := NewAutoScaler(5, 50, 10)
	if as.IsOn() {
		t.Errorf("AutoScaler should be off by default")
	}
	if as.Max == nil {
		t.Fatalf("Max is nil")
	}
	if as.Threshold == nil {
		t.Fatalf("Threshold is nil")
	}
	if as.baseline == nil {
		t.Fatalf("old is nil")
	}
	if as.state != stateDisabled {
		t.Fatalf("state is not disabled")
	}
	if as.ScaleAnts(dummyPool, 0, 0) {
		t.Fatalf("ScaleAnts should return false")
	}
	if as.ScaleAnts(dummyPool, 0, 1) {
		t.Fatalf("ScaleAnts should return false")
	}
	if as.ScaleAnts(dummyPool, 10, 9) {
		t.Fatalf("ScaleAnts should return false")
	}
	as.Enable()
	if !as.IsOn() {
		t.Fatalf("AutoScaler should be on")
	}
	if as.state != stateIdle {
		t.Fatalf("state is not idle")
	}
	if !as.ScaleAnts(dummyPool, 10, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if as.state != stateScalingUp {
		t.Fatalf("state is not scaling up")
	}
	if dummyPool.Cap() != 6 {
		t.Fatalf("Pool cap is not 6")
	}
	if !as.ScaleAnts(dummyPool, 11, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if dummyPool.Cap() != 7 {
		t.Fatalf("Pool cap is not 7")
	}
	if !as.ScaleAnts(dummyPool, 12, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if dummyPool.Cap() != 8 {
		t.Fatalf("Pool cap is not 8")
	}
	if !as.ScaleAnts(dummyPool, 13, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if dummyPool.Cap() != 9 {
		t.Fatalf("Pool cap is not 9")
	}
	if !as.ScaleAnts(dummyPool, 21, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if dummyPool.Cap() != 8 {
		t.Fatalf("Pool cap is not 8")
	}
	if !as.ScaleAnts(dummyPool, 21, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if dummyPool.Cap() != 7 {
		t.Fatalf("Pool cap is not 7")
	}
	if !as.ScaleAnts(dummyPool, 21, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if dummyPool.Cap() != 6 {
		t.Fatalf("Pool cap is not 6")
	}
	if !as.ScaleAnts(dummyPool, 21, 9) {
		t.Fatalf("ScaleAnts should return true")
	}
	if dummyPool.Cap() != 5 {
		t.Fatalf("Pool cap is not 5")
	}
	if as.ScaleAnts(dummyPool, 21, 9) {
		t.Fatalf("ScaleAnts should return false")
	}
	if dummyPool.Cap() != 5 {
		t.Fatalf("Pool cap is not 5")
	}
}
