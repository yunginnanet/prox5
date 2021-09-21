package pxndscvm

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	ipa "inet.af/netaddr"
)


// throw shit proxies here, get map
var inChan chan string

func init() {
	inChan = make(chan string, 100000)
}

func (s *Swamp) stage1(in string) bool {
	if !strings.Contains(in, ":") {
		return false
	}
	split := strings.Split(in, ":")
	if _, err := ipa.ParseIP(split[0]); err != nil {
		return false
	}
	if _, err := strconv.Atoi(split[1]); err != nil {
		return false
	}
	return true
}

// LoadProxyTXT loads proxies from a given seed file and randomly feeds them to the workers.
func (s *Swamp) LoadProxyTXT(seedFile string) int {
	var count int
	s.dbgPrint("LoadProxyTXT start: "+seedFile)
	defer s.dbgPrint("LoadProxyTXT finished: " + strconv.Itoa(count))

	f, err := os.Open(seedFile)
	if err != nil {
		return 0
	}

	scan := bufio.NewScanner(f)

	for scan.Scan() {
		if !s.stage1(scan.Text()) {
			continue
		}
		go s.LoadSingleProxy(scan.Text())
		count++
	}

	if err := f.Close(); err != nil {
		s.dbgPrint(err.Error())
		return count
	}

	return count
}

// LoadSingleProxy loads a SOCKS proxy into our queue as the format: 127.0.0.1:1080 (host:port)
func (s *Swamp) LoadSingleProxy(sock string) {
	inChan <- sock
}

// LoadMultiLineString loads a multiine string object with one (host:port) SOCKS proxy per line
func (s *Swamp) LoadMultiLineString(socks string) int {
	var count int
	scan := bufio.NewScanner(strings.NewReader(socks))
	for scan.Scan() {
		go s.LoadSingleProxy(scan.Text())
		count++
	}
	if count < 1 {
		return 0
	}
	return count
}

// ClearSOCKSList clears the slice of proxies that we continually draw from at random for validation
//	* Other operations (proxies that are still in buffered channels) will resume unless paused.
func (s *Swamp) ClearSOCKSList() {
	s.swampmap.clear()
}
