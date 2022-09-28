package prox5

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

// LoadProxyTXT loads proxies from a given seed file and feeds them to the mapBuilder to be later queued automatically for validation.
// Expects one of the following formats for each line:
//   - 127.0.0.1:1080
//   - 127.0.0.1:1080:user:pass
//   - yeet.com:1080
//   - yeet.com:1080:user:pass
//   - [fe80::2ef0:5dff:fe7f:c299]:1080
//   - [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (p5 *Swamp) LoadProxyTXT(seedFile string) (count int) {
	f, err := os.Open(seedFile)
	if err != nil {
		p5.dbgPrint(simpleString(err.Error()))
		return 0
	}

	defer func() {
		if err := f.Close(); err != nil {
			p5.dbgPrint(simpleString(err.Error()))
		}
	}()

	bs, err := io.ReadAll(f)
	if err != nil {
		p5.dbgPrint(simpleString(err.Error()))
		return 0
	}
	sockstr := string(bs)

	return p5.LoadMultiLineString(sockstr)
}

// LoadSingleProxy loads a SOCKS proxy into our map.
// Expects one of the following formats:
//   - 127.0.0.1:1080
//   - 127.0.0.1:1080:user:pass
//   - yeet.com:1080
//   - yeet.com:1080:user:pass
//   - [fe80::2ef0:5dff:fe7f:c299]:1080
//   - [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (p5 *Swamp) LoadSingleProxy(sock string) bool {
	var ok bool
	if sock, ok = filter(sock); !ok {
		return false
	}
	if err := p5.loadSingleProxy(sock); err != nil {
		return false
	}
	return true
}

func (p5 *Swamp) loadSingleProxy(sock string) error {
	if _, ok := p5.swampmap.add(sock); !ok {
		return errors.New("proxy already exists")
	}
	return nil
}

// LoadMultiLineString loads a multiine string object with proxy per line.
// Expects one of the following formats for each line:
//   - 127.0.0.1:1080
//   - 127.0.0.1:1080:user:pass
//   - yeet.com:1080
//   - yeet.com:1080:user:pass
//   - [fe80::2ef0:5dff:fe7f:c299]:1080
//   - [fe80::2ef0:5dff:fe7f:c299]:1080:user:pass
func (p5 *Swamp) LoadMultiLineString(socks string) int {
	var count int
	scan := bufio.NewScanner(strings.NewReader(socks))
	for scan.Scan() {
		if err := p5.loadSingleProxy(scan.Text()); err != nil {
			continue
		}
		count++
	}
	return count
}

// ClearSOCKSList clears the map of proxies that we have on record.
// Other operations (proxies that are still in buffered channels) will continue.
func (p5 *Swamp) ClearSOCKSList() {
	p5.swampmap.clear()
}
