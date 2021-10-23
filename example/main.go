package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mattn/go-tty"

	"git.tcp.direct/kayos/Prox5"
)

var swamp *Prox5.Swamp
var quit chan bool

func init() {
	quit = make(chan bool)
	swamp = Prox5.NewDefaultSwamp()
	swamp.SetMaxWorkers(5)
	swamp.EnableDebug()

	count := swamp.LoadProxyTXT("socks.list")
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
	t, err := tty.Open()
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

	t.Close()
	quit <- true
	return
}

func main() {
	go watchKeyPresses()

	go func() {
		for {
			fmt.Printf("4: %d, 4a: %d, 5: %d \n", swamp.Stats.Valid4, swamp.Stats.Valid4a, swamp.Stats.Valid5)
			time.Sleep(5 * time.Second)
		}
	}()

	<-quit
}
