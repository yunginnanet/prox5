package prox5

import (
	"net/netip"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

func filterv6(in string) (filtered string, ok bool) {
	split := strings.Split(in, "]:")
	if len(split) < 2 {

		return "", false
	}
	split2 := strings.Split(split[1], ":")
	switch len(split2) {
	case 0:
		combo, err := netip.ParseAddrPort(buildProxyString("", "", split[0], split2[0], true))
		if err == nil {
			return combo.String(), true
		}
	case 1:
		concat := buildProxyString("", "", split[0], split2[0], true)
		combo, err := netip.ParseAddrPort(concat)
		if err == nil {
			return combo.String(), true
		}
	default:
		_, err := netip.ParseAddrPort(buildProxyString("", "", split[0], split2[0], true))
		if err == nil {
			return buildProxyString(split2[1], split2[2], split[0], split2[0], true), true
		}
	}
	return "", true
}

func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func buildProxyString(username, password, address, port string, v6 bool) (result string) {
	builder := strs.Get()
	defer strs.MustPut(builder)
	if username != "" && password != "" {
		builder.MustWriteString(username)
		builder.MustWriteString(":")
		builder.MustWriteString(password)
		builder.MustWriteString("@")
	}
	builder.MustWriteString(strings.ToLower(address))
	if v6 {
		builder.MustWriteString("]")
	}
	builder.MustWriteString(":")
	builder.MustWriteString(port)
	return builder.String()
}

func filter(in string) (filtered string, ok bool) { //nolint:cyclop
	protoStr, protoNormalized, protoOK := protoStrNormalize(in)
	if protoOK {
		in = protoNormalized
	}

	split := strings.Split(in, ":")

	if !strings.Contains(in, ":") {
		in = in + ":1080"
	}

	if len(split) < 2 {
		return "", false
	}
	switch len(split) {
	case 2:
		_, isDomain := dns.IsDomainName(split[0])
		if isDomain && isNumber(split[1]) {
			return protoStr + strings.ToLower(in), true
		}
		combo, err := netip.ParseAddrPort(in)
		if err != nil {
			return "", false
		}
		return combo.String(), true
	case 3:
		if !strings.Contains(in, "@") {
			return "", false
		}
		domSplit := strings.Split(in, "@")
		if !strings.Contains(domSplit[0], ":") {
			return "", false
		}
		splitAuth := strings.Split(domSplit[0], ":")
		splitServ := strings.Split(domSplit[1], ":")
		_, isDomain := dns.IsDomainName(splitServ[0])
		if isDomain && isNumber(splitServ[1]) {
			return protoStr + buildProxyString(splitAuth[0], splitAuth[1],
				splitServ[0], splitServ[1], false), true
		}
		if _, err := netip.ParseAddrPort(domSplit[1]); err == nil {
			return protoStr + buildProxyString(splitAuth[0], splitAuth[1],
				splitServ[0], splitServ[1], false), true
		}
	case 4:
		_, isDomain := dns.IsDomainName(split[0])
		if isDomain && isNumber(split[1]) {
			return protoStr + buildProxyString(split[2], split[3], split[0], split[1], false), true
		}
		_, isDomain = dns.IsDomainName(split[2])
		if isDomain && isNumber(split[3]) {
			return protoStr + buildProxyString(split[0], split[1], split[2], split[3], false), true
		}
		if _, err := netip.ParseAddrPort(split[2] + ":" + split[3]); err == nil {
			return protoStr + buildProxyString(split[0], split[1], split[2], split[3], false), true
		}
		if _, err := netip.ParseAddrPort(split[0] + ":" + split[1]); err == nil {
			return protoStr + buildProxyString(split[2], split[3], split[0], split[1], false), true
		}
	default:
		if !strings.Contains(in, "[") || !strings.Contains(in, "]:") {
			return "", false
		}
	}
	v6Filt, v6Ok := filterv6(in)
	if v6Ok {
		v6Filt = protoStr + v6Filt
	}
	return v6Filt, v6Ok
}
