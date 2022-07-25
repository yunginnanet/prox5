package prox5

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
)

// GetProto retrieves the known protocol value of the Proxy.
func (sock *Proxy) GetProto() ProxyProtocol {
	return sock.proto
}

// GetProto safely retrieves the protocol value of the Proxy.
func (sock *Proxy) String() string {
	tout := ""
	if sock.parent.GetServerTimeoutStr() != "-1" {
		tbuf := copABuffer.Get().(*strings.Builder)
		tbuf.WriteString("?timeout=")
		tbuf.WriteString(sock.parent.GetServerTimeoutStr())
		tbuf.WriteString("s")
		tout = tbuf.String()
		discardBuffer(tbuf)
	}
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("socks")
	buf.WriteString(getProtoStr(sock.GetProto()))
	buf.WriteString("://")
	buf.WriteString(sock.Endpoint)
	if tout != "" {
		buf.WriteString(tout)
	}
	out := buf.String()
	discardBuffer(buf)
	return out
}

// GetStatistics returns all current statistics.
// * This is a pointer, do not modify it!
func (pe *ProxyEngine) GetStatistics() *statistics {
	return pe.stats
}

// RandomUserAgent retrieves a random user agent from our list in string form.
func (pe *ProxyEngine) RandomUserAgent() string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return entropy.RandomStrChoice(pe.swampopt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our ProxyEngine's options
func (pe *ProxyEngine) GetRandomEndpoint() string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return entropy.RandomStrChoice(pe.swampopt.checkEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (pe *ProxyEngine) GetStaleTime() time.Duration {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.stale
}

// GetValidationTimeout returns the current value of validationTimeout.
func (pe *ProxyEngine) GetValidationTimeout() time.Duration {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.validationTimeout
}

// GetValidationTimeoutStr returns the current value of validationTimeout (in seconds string).
func (pe *ProxyEngine) GetValidationTimeoutStr() string {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	timeout := pe.swampopt.validationTimeout
	return strconv.Itoa(int(timeout / time.Second))
}

// GetServerTimeout returns the current value of serverTimeout.
func (pe *ProxyEngine) GetServerTimeout() time.Duration {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.serverTimeout
}

// GetServerTimeoutStr returns the current value of serverTimeout (in seconds string).
func (pe *ProxyEngine) GetServerTimeoutStr() string {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	timeout := pe.swampopt.serverTimeout
	if timeout == time.Duration(0) {
		return "-1"
	}
	return strconv.Itoa(int(timeout / time.Second))
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (pe *ProxyEngine) GetMaxWorkers() int {
	return pe.pool.Cap()
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (pe *ProxyEngine) IsRunning() bool {
	return atomic.LoadInt32(&pe.runningdaemons) == 2
}

// GetRecyclingStatus retrieves the current recycling status, see EnableRecycling.
func (pe *ProxyEngine) GetRecyclingStatus() bool {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.recycle
}

// GetWorkers retrieves pond worker statistics:
//    * return MaxWorkers, RunningWorkers, IdleWorkers
func (pe *ProxyEngine) GetWorkers() (maxWorkers, runningWorkers, idleWorkers int) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.pool.Cap(), pe.pool.Running(), pe.pool.Free()
}

// GetRemoveAfter retrieves the removeafter policy, the amount of times a recycled proxy is marked as bad until it is removed entirely.
//    *  returns -1 if recycling is disabled.
func (pe *ProxyEngine) GetRemoveAfter() int {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	if !pe.swampopt.recycle {
		return -1
	}
	return pe.swampopt.removeafter
}

// GetDialerBailout retrieves the dialer bailout policy. See SetDialerBailout for more info.
func (pe *ProxyEngine) GetDialerBailout() int {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.swampopt.dialerBailout
}

// TODO: Document middleware concept

func (pe *ProxyEngine) GetDispenseMiddleware() func(*Proxy) (*Proxy, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.dispenseMiddleware
}
