package pxndscvm

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	ipa "inet.af/netaddr"
)

// throw shit proxies here, get map
// see daemons.go
var inChan chan string

func init() {
	inChan = make(chan string, 100000)
}

func (s *Swamp) stage1(in string) (string, bool) {
	if !strings.Contains(in, ":") {
		return in, false
	}
	split := strings.Split(in, ":")
	if _, err := ipa.ParseIP(split[0]); err != nil {
		return in, false
	}
	if _, err := strconv.Atoi(split[1]); err != nil {
		return in, false
	}
	return fmt.Sprintf("%s:%s", split[0], split[1]), true
}

// LoadProxyTXT loads proxies from a given seed file and feeds them to the mapBuilder to be later queued automatically for validation.
func (s *Swamp) LoadProxyTXT(seedFile string) int {
	var count int
	var filtered string
	var ok bool
	s.dbgPrint("LoadProxyTXT start: " + seedFile)
	defer s.dbgPrint("LoadProxyTXT finished: " + strconv.Itoa(count))

	f, err := os.Open(seedFile)
	if err != nil {
		return 0
	}

	scan := bufio.NewScanner(f)

	for scan.Scan() {
		if filtered, ok = s.stage1(scan.Text()); !ok {
			continue
		}
		go s.LoadSingleProxy(filtered)
		count++
	}

	if err := f.Close(); err != nil {
		s.dbgPrint(err.Error())
		return count
	}

	return count
}

// LoadSingleProxy loads a SOCKS proxy into our map. Uses the format: 127.0.0.1:1080 (host:port).
func (s *Swamp) LoadSingleProxy(sock string) {
	inChan <- sock
}

// LoadMultiLineString loads a multiine string object with one (host:port) SOCKS proxy per line.
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

// ClearSOCKSList clears the map of proxies that we have on record.
// Other operations (proxies that are still in buffered channels) will continue unless paused.
func (s *Swamp) ClearSOCKSList() {
	s.swampmap.clear()
}
