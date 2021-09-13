package pxndscvm

import "sync"

// Statistics is used to encapsulate various swampy stats
type Statistics struct {
	validated4 int
	validated4a int
	validated5 int

	mu *sync.Mutex
}

func (stats *Statistics) v4() {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.validated4++
}

func (stats *Statistics) v4a() {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.validated4a++
}

func (stats *Statistics) v5() {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.validated5++
}
