package pxndscvm

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

func (s *Swamp) EnableDebug() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.Debug = true
}

func (s *Swamp) DisableDebug() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.Debug = false
}
