package prox5

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	ipa "inet.af/netaddr"
)

// throw shit proxies here, get map
// see daemons.go
var inChan chan string

func init() {
	inChan = make(chan string, 100000)
}

func checkV6(in string) (filtered string, ok bool) {
	split := strings.Split(in, "]:")
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

func (pe *ProxyEngine) filter(in string) (filtered string, ok bool) {
	if !strings.Contains(in, ":") {
		return in, false
	}
	split := strings.Split(in, ":")

	if len(split) < 2 {
		return in, false
	}

	if _, err := strconv.Atoi(split[1]); err != nil {
		return in, false
	}

	switch len(split) {
	case 2:
		if _, ok := dns.IsDomainName(split[0]); ok {
			return in, true
		}
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
		if !strings.Contains(split[0], "[") || !strings.Contains(split[0], "]:") {
			return in, false
		}
	}
	return checkV6(in)
}

// LoadProxyTXT loads proxies from a given seed file and feeds them to the mapBuilder to be later queued automatically for validation.
// Expects the following formats:
// * 127.0.0.1:1080
// * 127.0.0.1:1080:user:pass
// * yeet.com:1080
// * yeet.com:1080:user:pass
// * [fe80::2ef0:5dff:fe7f:c299]:1080
// * [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (pe *ProxyEngine) LoadProxyTXT(seedFile string) int {
	var count = &atomic.Value{}
	count.Store(0)

	f, err := os.Open(seedFile)
	if err != nil {
		pe.dbgPrint(err.Error())
		return 0
	}

	pe.dbgPrint("LoadProxyTXT start: " + seedFile)
	defer func() {
		pe.dbgPrint("LoadProxyTXT finished: " + strconv.Itoa(count.Load().(int)))
		if err := f.Close(); err != nil {
			pe.dbgPrint(err.Error())
		}
	}()

	bs, err := io.ReadAll(f)
	if err != nil {
		pe.dbgPrint(err.Error())
		return 0
	}
	sockstr := string(bs)

	count.Store(pe.LoadMultiLineString(sockstr))
	return count.Load().(int)
}

// LoadSingleProxy loads a SOCKS proxy into our map. Uses the format: 127.0.0.1:1080 (host:port).
func (pe *ProxyEngine) LoadSingleProxy(sock string) (ok bool) {
	if sock, ok = pe.filter(sock); !ok {
		return
	}
	go pe.loadSingleProxy(sock)
	return
}

func (pe *ProxyEngine) loadSingleProxy(sock string) {
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
func (pe *ProxyEngine) LoadMultiLineString(socks string) int {
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
func (pe *ProxyEngine) ClearSOCKSList() {
	pe.swampmap.clear()
}
