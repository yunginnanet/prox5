package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"git.tcp.direct/kayos/prox5"
	"github.com/haxii/socks5"
	"github.com/mattn/go-tty"
)

var (
	swamp *prox5.Swamp
	quit  chan bool
	t     *tty.TTY
)

type socksLogger struct{}

var socklog = socksLogger{}

// Printf is used to handle socks server logging.
func (s socksLogger) Printf(format string, a ...interface{}) {
	println(fmt.Sprintf(format, a))
}

func StartUpstreamProxy(listen string) {
	conf := &socks5.Config{Dial: swamp.DialContext, Logger: socklog}
	server, err := socks5.New(conf)
	if err != nil {
		println(err.Error())
		return
	}

	socklog.Printf("starting proxy server on %s", listen)
	if err := server.ListenAndServe("tcp", listen); err != nil {
		println(err.Error())
		return
	}
}

func init() {
	quit = make(chan bool)
	swamp = prox5.NewDefaultSwamp()
	swamp.SetMaxWorkers(5)
	swamp.EnableDebug()
	go StartUpstreamProxy("127.0.0.1:1555")

	count := swamp.LoadProxyTXT(os.Args[1])
	if count < 1 {
		println("file contained no valid SOCKS host:port combos")
		os.Exit(1)
	}

	if err := swamp.Start(); err != nil {
		panic(err)
	}

	println("[USAGE] q: quit | d: debug | a: socks4 | b: socks4a | c: socks5 | p: pause/unpause")
}

func get(ver string) {
	switch ver {
	case "4":
		println("retrieving SOCKS4...")
		println(swamp.Socks4Str())
	case "4a":
		println("retrieving SOCKS4a...")
		println(swamp.Socks4aStr())
	case "5":
		println("retrieving SOCKS5...")
		println(swamp.Socks5Str())
	default:

	}
}

func watchKeyPresses() {
	var err error
	t, err = tty.Open()
	if err != nil {
		panic(err)
	}
	var done = false

	for {
		r, err := t.ReadRune()
		if err != nil {
			panic(err)
		}
		switch string(r) {
		case "d":
			if swamp.DebugEnabled() {
				println("disabling debug")
				swamp.DisableDebug()
			} else {
				println("enabling debug")
				swamp.EnableDebug()
			}
		case "+":
			swamp.SetMaxWorkers(swamp.GetMaxWorkers() + 1)
			println("New worker count: " + strconv.Itoa(swamp.GetMaxWorkers()))
		case "-":
			swamp.SetMaxWorkers(swamp.GetMaxWorkers() - 1)
			println("New worker count: " + strconv.Itoa(swamp.GetMaxWorkers()))
		case "a":
			go get("4")
		case "b":
			go get("4a")
		case "c":
			go get("5")
		case "p":
			if swamp.IsRunning() {
				err := swamp.Pause()
				if err != nil {
					println(err.Error())
				}
			} else {
				if err := swamp.Resume(); err != nil {
					println(err.Error())
				}
			}
		case "q":
			done = true
			break
		default:
			//
		}
		if done {
			break
		}
	}

	quit <- true
	return
}

func main() {
	go watchKeyPresses()

	defer func(t *tty.TTY) {
		_ = t.Close()
	}(t)

	go func() {
		for {
			fmt.Printf("4: %d, 4a: %d, 5: %d \n", swamp.Stats.Valid4, swamp.Stats.Valid4a, swamp.Stats.Valid5)
			time.Sleep(5 * time.Second)
		}
	}()

	<-quit
}
