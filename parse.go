package prox5

import (
	"strconv"
	"strings"

	"github.com/miekg/dns"
	ipa "inet.af/netaddr"

	"git.tcp.direct/kayos/prox5/internal/pools"
)

func filterv6(in string) (filtered string, ok bool) {
	split := strings.Split(in, "]:")
	if len(split) < 2 {

		return "", false
	}
	split2 := strings.Split(split[1], ":")
	switch len(split2) {
	case 0:
		combo, err := ipa.ParseIPPort(buildProxyString("", "", split[0], split2[0], true))
		if err == nil {
			return combo.String(), true
		}
	case 1:
		println("len(split2) == 1")
		concat := buildProxyString("", "", split[0], split2[0], true)
		combo, err := ipa.ParseIPPort(concat)
		if err == nil {
			return combo.String(), true
		}
	default:
		println("len(split2) != 1")
		_, err := ipa.ParseIPPort(buildProxyString("", "", split[0], split2[0], true))
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
	builder := pools.CopABuffer.Get().(*strings.Builder)
	if username != "" && password != "" {
		builder.WriteString(username)
		builder.WriteString(":")
		builder.WriteString(password)
		builder.WriteString("@")
	}
	builder.WriteString(address)
	if v6 {
		builder.WriteString("]")
	}
	builder.WriteString(":")
	builder.WriteString(port)
	result = builder.String()
	pools.DiscardBuffer(builder)
	return
}

func filter(in string) (filtered string, ok bool) { //nolint:cyclop
	if !strings.Contains(in, ":") {
		return "", false
	}
	split := strings.Split(in, ":")

	if len(split) < 2 {
		return "", false
	}
	switch len(split) {
	case 2:
		_, isDomain := dns.IsDomainName(split[0])
		if isDomain && isNumber(split[1]) {
			return in, true
		}
		combo, err := ipa.ParseIPPort(in)
		if err != nil {
			return "", false
		}
		return combo.String(), true
	case 4:
		_, isDomain := dns.IsDomainName(split[0])
		if isDomain && isNumber(split[1]) {
			return buildProxyString(split[2], split[3], split[0], split[1], false), true
		}
		_, isDomain = dns.IsDomainName(split[2])
		if isDomain && isNumber(split[3]) {
			return buildProxyString(split[0], split[1], split[2], split[3], false), true
		}
		if _, err := ipa.ParseIPPort(split[2] + ":" + split[3]); err == nil {
			return buildProxyString(split[0], split[1], split[2], split[3], false), true
		}
		if _, err := ipa.ParseIPPort(split[0] + ":" + split[1]); err == nil {
			return buildProxyString(split[2], split[3], split[0], split[1], false), true
		}
	default:
		if !strings.Contains(in, "[") || !strings.Contains(in, "]:") {
			return "", false
		}
	}
	return filterv6(in)
}
