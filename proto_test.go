package prox5

import (
	"testing"
)

func TestImmutableProto(t *testing.T) {
	prt := newImmutableProto()
	if prt.Get() != ProtoNull {
		t.Fatal("expected protonull")
	}
	prt.set(ProtoSOCKS5)
	if prt.Get() != ProtoSOCKS5 {
		t.Fatal("expected socks5 proto")
	}
	prt.set(ProtoSOCKS4)
	if prt.Get() != ProtoSOCKS5 {
		t.Fatal("expected socks5 proto still after trying to set twice")
	}
	str := strs.Get()
	defer strs.MustPut(str)
	prt.Get().writeProtoString(str)
	if str.String() != "socks5" {
		t.Fatalf("expected socks5://, got %s", str.String())
	}
	str.MustReset()
	prt.Get().writeProtoURI(str)
	if str.String() != "socks5://" {
		t.Fatalf("expected socks5://, got %s", str.String())
	}
	str.MustReset()
	ptrstr := prt.Get().String()
	if ptrstr != "socks5" {
		t.Fatalf("expected socks5://, got %s", ptrstr)
	}
}
