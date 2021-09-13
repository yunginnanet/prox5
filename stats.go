package pxndscvm

import "sync"

// Statistics is used to encapsulate various swampy stats
type Statistics struct {
	Valid4  int
	Valid4a int
	Valid5  int

	mu *sync.Mutex
}

func (stats *Statistics) v4() {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.Valid4++
}

func (stats *Statistics) v4a() {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.Valid4a++
}

func (stats *Statistics) v5() {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.Valid5++
}
