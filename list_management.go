package pxndscvm

import (
	"bufio"
	"os"
	"strings"
)

// LoadProxyTXT loads proxies from a given seed file and randomly feeds them to the workers.
// Call Start after this
func (s *Swamp) LoadProxyTXT(seedFile string) error {
	s.dbgPrint("LoadProxyTXT start")

	f, err := os.Open(seedFile)
	if err != nil {
		return err
	}

	scan := bufio.NewScanner(f)

	for scan.Scan() {
		s.mu.Lock()
		s.scvm = append(s.scvm, scan.Text())
		s.mu.Unlock()
	}

	if err := f.Close(); err != nil {
		s.dbgPrint(err.Error())
		return err
	}
	return nil
}

// LoadSingleProxy loads a SOCKS proxy into our queue as the format: 127.0.0.1:1080 (host:port)
func (s *Swamp) LoadSingleProxy(sock string) {
	s.mu.Lock()
	s.scvm = append(s.scvm, sock)
	s.mu.Unlock()
}

// LoadMultiLineString loads a multiine string object with one (host:port) SOCKS proxy per line
func (s *Swamp) LoadMultiLineString(socks string) error {
	scan := bufio.NewScanner(strings.NewReader(socks))
	for scan.Scan() {
		s.LoadSingleProxy(scan.Text())
	}
	return nil
}

// ClearSOCKSList clears the slice of proxies that we continually draw from at random for validation
//	* Other operations (proxies that are still in buffered channels) will resume unless paused.
func (s *Swamp) ClearSOCKSList() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scvm = []string{}
}
