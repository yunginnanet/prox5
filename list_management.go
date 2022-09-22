package prox5

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"
)

// throw shit proxies here, get map
// see daemons.go
var inChan chan string

func init() {
	inChan = make(chan string, 100000)
}

// LoadProxyTXT loads proxies from a given seed file and feeds them to the mapBuilder to be later queued automatically for validation.
// Expects one of the following formats for each line:
// * 127.0.0.1:1080
// * 127.0.0.1:1080:user:pass
// * yeet.com:1080
// * yeet.com:1080:user:pass
// * [fe80::2ef0:5dff:fe7f:c299]:1080
// * [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (pe *Swamp) LoadProxyTXT(seedFile string) (count int) {
	f, err := os.Open(seedFile)
	if err != nil {
		pe.dbgPrint(simpleString(err.Error()))
		return 0
	}

	defer func() {
		if err := f.Close(); err != nil {
			pe.dbgPrint(simpleString(err.Error()))
		}
	}()

	bs, err := io.ReadAll(f)
	if err != nil {
		pe.dbgPrint(simpleString(err.Error()))
		return 0
	}
	sockstr := string(bs)

	return pe.LoadMultiLineString(sockstr)
}

// LoadSingleProxy loads a SOCKS proxy into our map.
// Expects one of the following formats:
// * 127.0.0.1:1080
// * 127.0.0.1:1080:user:pass
// * yeet.com:1080
// * yeet.com:1080:user:pass
// * [fe80::2ef0:5dff:fe7f:c299]:1080
// * [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (pe *Swamp) LoadSingleProxy(sock string) (ok bool) {
	if sock, ok = filter(sock); !ok {
		return
	}
	go pe.loadSingleProxy(sock)
	return
}

func (pe *Swamp) loadSingleProxy(sock string) {
	for {
		select {
		case inChan <- sock:
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

// LoadMultiLineString loads a multiine string object with proxy per line.
// Expects one of the following formats for each line:
// * 127.0.0.1:1080
// * 127.0.0.1:1080:user:pass
// * yeet.com:1080
// * yeet.com:1080:user:pass
// * [fe80::2ef0:5dff:fe7f:c299]:1080
// * [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (pe *Swamp) LoadMultiLineString(socks string) int {
	var count int
	scan := bufio.NewScanner(strings.NewReader(socks))
	for scan.Scan() {
		if pe.LoadSingleProxy(scan.Text()) {
			count++
		}
	}
	return count
}

// ClearSOCKSList clears the map of proxies that we have on record.
// Other operations (proxies that are still in buffered channels) will continue.
func (pe *Swamp) ClearSOCKSList() {
	pe.swampmap.clear()
}
