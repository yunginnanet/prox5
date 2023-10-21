package prox5

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.tcp.direct/kayos/common/entropy"
	"git.tcp.direct/kayos/go-socks5"
)

func init() {
	os.Setenv("PROX5_SCALER_DEBUG", "1")
}

var failures int64 = 0

type randomFail struct {
	t           *testing.T
	failedCount int64
	maxFail     int64

	failOneOutOf int
}

func (rf *randomFail) fail() bool {
	if rf.failOneOutOf == 0 {
		return false
	}

	doFail := entropy.GetOptimizedRand().Intn(rf.failOneOutOf) == 1

	if !doFail {
		return false
	}
	atomic.AddInt64(&rf.failedCount, 1)
	rf.t.Logf("random SOCKS failure triggered, total fail count: %d", rf.failedCount)
	if rf.maxFail > 0 && atomic.LoadInt64(&rf.failedCount) > rf.maxFail {
		rf.t.Errorf("[FAIL] random SOCKS failure triggered too many times, total fail count: %d", rf.failedCount)
	}

	atomic.AddInt64(&failures, 1)
	return true
}

type dummyHTTPServer struct {
	t *testing.T
	net.Listener
}

func timeNowJSON() []byte {
	js, _ := time.Now().MarshalJSON()
	return js
}

func newDummyHTTPSServer(t *testing.T, port int) {
	t.Helper()
	dtcp := &dummyHTTPServer{t: t}
	var err error
	if dtcp.Listener, err = net.Listen("tcp", ":"+strconv.Itoa(port)); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
	go func() {
		if err = http.Serve(dtcp, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Duration(entropy.RNG(300)) * time.Millisecond)
			if _, err = w.Write(timeNowJSON()); err != nil {
				t.Error("[FAIL] http server failed to write JSON: " + err.Error())
			}
		})); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Error("[FAIL] http.Serve error: " + err.Error())
		}
	}()

	t.Cleanup(func() {
		_ = dtcp.Close()
	})

	t.Logf("dummy HTTPS server listening on port %d", port)

}

var ErrRandomFail = errors.New("random failure")

func dummySOCKSServer(t *testing.T, port int, rf ...*randomFail) {
	t.Helper()
	var failure = &randomFail{t: t, failedCount: int64(0), failOneOutOf: 0}
	if len(rf) > 0 {
		failure = rf[0]
	}

	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if failure.fail() {
			return nil, ErrRandomFail
		}
		time.Sleep(time.Duration(entropy.GetOptimizedRand().Intn(300)) * time.Millisecond)
		return net.Dial(network, addr)
	}

	server := socks5.NewServer(socks5.WithDial(dialer))
	go func() {
		err := server.ListenAndServe("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err != nil && !errors.Is(err, net.ErrClosed) {
			t.Error("[FAIL] socks server failure: " + err.Error())
		}
	}()
}

type p5TestLogger struct {
	t *testing.T
}

func (tl p5TestLogger) Errorf(format string, args ...interface{}) {
	tl.t.Logf("[ERROR] "+format, args...)
}
func (tl p5TestLogger) Printf(format string, args ...interface{}) {
	val := fmt.Sprintf(format, args...)
	if strings.Contains(val, "failed to verify") {
		atomic.AddInt64(&failures, 1)
	}
	tl.t.Logf("[PRINT] " + val)
}
func (tl p5TestLogger) Print(args ...interface{}) {
	val := fmt.Sprintf("%+v", args...)
	if strings.Contains(val, "failed to verify") {
		atomic.AddInt64(&failures, 1)
	}
	tl.t.Log("[PRINT] " + val)
}
func TestProx5(t *testing.T) {
	numTest := 100
	if envCount := os.Getenv("PROX5_TEST_COUNT"); envCount != "" {
		n, e := strconv.Atoi(envCount)
		if e != nil {
			t.Skip(e.Error())
		}
		numTest = n
	}
	for i := 0; i < numTest; i++ {
		dummySOCKSServer(t, 5555+i, &randomFail{
			t:            t,
			failedCount:  int64(0),
			failOneOutOf: entropy.RNG(200),
			maxFail:      50,
		})
		time.Sleep(time.Millisecond * 5)
	}
	newDummyHTTPSServer(t, 8055)
	time.Sleep(time.Millisecond * 350)
	p5 := NewProxyEngine()
	p5.SetAndEnableDebugLogger(p5TestLogger{t: t})
	p5.SetMaxWorkers(10)
	p5.EnableAutoScaler()
	p5.SetAutoScalerThreshold(10)
	// p5.SetValidationTimeout(200 * time.Millisecond)
	p5.SetAutoScalerMaxScale(100)
	// p5.DisableRecycling()
	p5.SetRemoveAfter(2)
	var index = 5555

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	var once = &sync.Once{}

	check5 := func() {
		if err := p5.Pause(); err != nil {
			t.Errorf("[FAIL] failed to pause: %s", err.Error())
		}
		time.Sleep(time.Second * 1)
		got := p5.GetTotalValidated()
		want := 55 - int(atomic.LoadInt64(&failures))
		if got != want {
			t.Logf("[WARN] total validated proxies does not match expected, got: %d, expected: %d",
				got, want)
		}
		if err := p5.Resume(); err != nil {
			t.Errorf("[FAIL] failed to resume: %s", err.Error())
		}
	}

	load := func() {
		if index > 5555+numTest {
			return
		}
		entropy.RandSleepMS(150)
		p5.LoadSingleProxy("127.0.0.1:" + strconv.Itoa(index))
		if index == 5555+55 {
			once.Do(check5)
		}
		index++
	}

	var successCount int64 = 0

	makeReq := func() {
		select {
		case <-ctx.Done():
			return
		default:

		}
		resp, err := p5.GetHTTPClient().Get("http://127.0.0.1:8055")
		if err != nil && !errors.Is(err, ErrNoProxies) && !errors.Is(err, net.ErrClosed) {
			t.Error("[FAIL] " + err.Error())
		}
		if err != nil && errors.Is(err, ErrNoProxies) {
			return
		}
		if resp == nil {
			return
		}
		b, e := io.ReadAll(resp.Body)
		if e != nil && !errors.Is(e, net.ErrClosed) {
			t.Log("[WARN] " + e.Error())
		}
		t.Logf("got proxied response: %s", string(b))
		atomic.AddInt64(&successCount, 1)
	}

	ticker := time.NewTicker(time.Millisecond * 100)

	if err := p5.Start(); err != nil {
		t.Fatal(err)
	}

	wait := 0

testLoop:
	for {
		select {
		case <-ctx.Done():
			successCountFinal := atomic.LoadInt64(&successCount)
			if successCountFinal < 10 {
				t.Fatal("no successful requests")
			}
			t.Logf("total successful requests: %d", successCountFinal)
			break testLoop
		case <-ticker.C:
			// pre-warm
			wait++
			if wait >= 50 {
				go makeReq()
			}
		default:
			load()
		}
	}
	cancel()
	if err := p5.Close(); err != nil {
		t.Fatal(err)
	}
	// let the proxy engine close gracefully
	time.Sleep(time.Second * 5)
}
