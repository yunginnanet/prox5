package pxndscvm

import "errors"

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (s *Swamp) AddUserAgents(uagents []string) {
	// mutex lock so that RLock during proxy checking will block while we change this value
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.UserAgents = append(s.swampopt.UserAgents, uagents...)
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (s *Swamp) SetUserAgents(uagents []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.UserAgents = append(s.swampopt.UserAgents, uagents...)
}

// EnableDebug enables printing of verbose messages during operation
func (s *Swamp) EnableDebug() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.Debug = true
}

// DisableDebug enables printing of verbose messages during operation
func (s *Swamp) DisableDebug() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.Debug = false
}

// SetMaxWorkers set the maximum workers for proxy checking, this must be set before calling LoadProxyTXT for the first time.
func (s *Swamp) SetMaxWorkers(num int) error {
	if s.started {
		return errors.New("can't change max workers during proxypool operation, only before")
	}
	s.swampopt.MaxWorkers = num
	return nil
}
