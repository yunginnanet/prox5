module git.tcp.direct/kayos/prox5

go 1.19

require (
	git.tcp.direct/kayos/common v0.8.0
	git.tcp.direct/kayos/go-socks5 v0.0.0-20221017013929-681914fcfad0
	git.tcp.direct/kayos/socks v0.0.0-20221016092341-00cb75cc613a
	github.com/gdamore/tcell/v2 v2.5.4
	github.com/miekg/dns v1.1.50
	github.com/ooni/oohttp v0.5.0
	github.com/orcaman/concurrent-map/v2 v2.0.1
	github.com/panjf2000/ants/v2 v2.7.1
	github.com/refraction-networking/utls v1.2.0
	github.com/rivo/tview v0.0.0-20221221172820-02e38ea9604c
	github.com/yunginnanet/Rate5 v1.2.1
	golang.org/x/net v0.5.0
	inet.af/netaddr v0.0.0-20220811202034-502d2d690317
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/klauspost/compress v1.15.12 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/rivo/uniseg v0.4.2 // indirect
	go4.org/intern v0.0.0-20211027215823-ae77deb06f29 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20220617031537-928513b29760 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/term v0.4.0 // indirect
	golang.org/x/text v0.6.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
	nullprogram.com/x/rng v1.1.0 // indirect
)

retract (
	v1.2.2 // cleanup
	v1.2.1 // accident
)
