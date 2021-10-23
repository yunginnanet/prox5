package Prox5

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/miekg/dns"
	ipa "inet.af/netaddr"
)

// throw shit proxies here, get map
// see daemons.go
var inChan chan string

func init() {
	inChan = make(chan string, 100000)
}

func filter(in string) (filtered string, ok bool) {
	if !strings.Contains(in, ":") {
		return in, false
	}

	split := strings.Split(in, ":")
	switch len(split) {
	case 1:
		return in, false
	case 2:
		combo, err := ipa.ParseIPPort(in)
		if err != nil {
			return in, false
		}
		return combo.String(), true
	case 4:
		if _, ok := dns.IsDomainName(split[0]); ok {
			return fmt.Sprintf("%s:%s@%s:%s", split[2], split[3], split[0], split[1]), true
		}
		combo, err := ipa.ParseIPPort(split[0] + ":" + split[1])
		if err != nil {
			return in, false
		}
		return fmt.Sprintf("%s:%s@%s", split[2], split[3], combo.String()), true
	default:
		if !strings.Contains(split[0], "[") || !strings.Contains(split[0], "]") {
			return in, false
		}
	}

	split = strings.Split(in, "]:")
	if len(split) != 2 {
		return in, false
	}

	combo, err := ipa.ParseIPPort(split[0] + "]:" + split[1])
	if err != nil {
		return in, false
	}

	if !strings.Contains(split[1], ":") {
		return combo.String(), true
	}

	split6 := strings.Split(split[1], ":")
	if len(split6) != 2 {
		return in, false
	}

	return fmt.Sprintf("%s:%s@%s", split6[0], split6[1], combo.String()), true
}

// LoadProxyTXT loads proxies from a given seed file and feeds them to the mapBuilder to be later queued automatically for validation.
func (s *Swamp) LoadProxyTXT(seedFile string) int {
	var count = &atomic.Value{}
	count.Store(0)
	var ok bool

	s.dbgPrint("LoadProxyTXT start: " + seedFile)
	defer func() {
		s.dbgPrint("LoadProxyTXT finished: " + strconv.Itoa(count.Load().(int)))
	}()

	f, err := os.Open(seedFile)
	if err != nil {
		s.dbgPrint(red + err.Error() + rst)
		return 0
	}

	scan := bufio.NewScanner(f)

	for scan.Scan() {
		if _, ok = filter(scan.Text()); !ok {
			continue
		}

		count.Store(count.Load().(int) + 1)
		go s.LoadSingleProxy(scan.Text())
	}

	if err := f.Close(); err != nil {
		s.dbgPrint(red + err.Error() + rst)
	}

	return count.Load().(int)
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
// Other operations (proxies that are still in buffered channels) will continue.
func (s *Swamp) ClearSOCKSList() {
	s.swampmap.clear()
}
