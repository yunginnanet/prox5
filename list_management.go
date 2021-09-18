package pxndscvm

import (
	"bufio"
	"errors"
	ipa "inet.af/netaddr"
	"os"
	"strconv"
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
func (s *Swamp) LoadSingleProxy(sock string) error {
	if !strings.Contains(sock, ":") {
		return errors.New("missing colon/missing port")
	}
	split := strings.Split(sock, ":")
	if _, err := ipa.ParseIP(split[0]); err != nil {
		return errors.New(split[0] + "is not an IP address")
	}
	if _, err := strconv.Atoi(split[1]); err != nil {
		return errors.New(split[1] + "is not a number")
	}
	s.mu.Lock()
	s.scvm = append(s.scvm, sock)
	s.mu.Unlock()
	return nil
}

// LoadMultiLineString loads a multiine string object with one (host:port) SOCKS proxy per line
func (s *Swamp) LoadMultiLineString(socks string) (int, error) {
	var count int
	scan := bufio.NewScanner(strings.NewReader(socks))
	for scan.Scan() {
		if err := s.LoadSingleProxy(scan.Text()); err == nil {
			count++
		}
	}
	if count < 1 {
		return 0, errors.New("no valid host:ip entries found in string")
	}
	return count, nil
}

// ClearSOCKSList clears the slice of proxies that we continually draw from at random for validation
//	* Other operations (proxies that are still in buffered channels) will resume unless paused.
func (s *Swamp) ClearSOCKSList() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scvm = []string{}
}
