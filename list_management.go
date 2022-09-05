package prox5

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// throw shit proxies here, get map
// see daemons.go
var inChan chan string

func init() {
	inChan = make(chan string, 100000)
}

// LoadProxyTXT loads proxies from a given seed file and feeds them to the mapBuilder to be later queued automatically for validation.
// Expects the following formats:
//    * 127.0.0.1:1080
//    * 127.0.0.1:1080:user:pass
//    * yeet.com:1080
//    * yeet.com:1080:user:pass
//    * [fe80::2ef0:5dff:fe7f:c299]:1080
//    * [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (s *Swamp) LoadProxyTXT(seedFile string) int {
	var count = &atomic.Value{}
	count.Store(0)

	f, err := os.Open(seedFile)
	if err != nil {
		s.dbgPrint(red + err.Error() + rst)
		return 0
	}

	s.dbgPrint("LoadProxyTXT start: " + seedFile)
	defer func() {
		s.dbgPrint("LoadProxyTXT finished: " + strconv.Itoa(count.Load().(int)))
		if err := f.Close(); err != nil {
			s.dbgPrint(red + err.Error() + rst)
		}
	}()

	bs, err := io.ReadAll(f)
	if err != nil {
		s.dbgPrint(red + err.Error() + rst)
		return 0
	}
	sockstr := string(bs)

	count.Store(s.LoadMultiLineString(sockstr))
	return count.Load().(int)
}

// LoadSingleProxy loads a SOCKS proxy into our map. Uses the format: 127.0.0.1:1080 (host:port).
func (s *Swamp) LoadSingleProxy(sock string) (ok bool) {
	if sock, ok = filter(sock); !ok {
		return
	}
	go s.loadSingleProxy(sock)
	return
}

func (s *Swamp) loadSingleProxy(sock string) {
	for {
		select {
		case inChan <- sock:
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

// LoadMultiLineString loads a multiine string object with one (host:port) SOCKS proxy per line.
func (s *Swamp) LoadMultiLineString(socks string) int {
	var count int
	scan := bufio.NewScanner(strings.NewReader(socks))
	for scan.Scan() {
		if s.LoadSingleProxy(scan.Text()) {
			count++
		}
	}
	return count
}

// ClearSOCKSList clears the map of proxies that we have on record.
// Other operations (proxies that are still in buffered channels) will continue.
func (s *Swamp) ClearSOCKSList() {
	s.swampmap.clear()
}
