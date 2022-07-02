package coredns_nftables

import (
	"testing"

	"github.com/coredns/caddy"
)

func TestSetup(t *testing.T) {
	c := caddy.NewTestController("dns", `nftables-response`)
	if err := setup(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `nftables-response example.org`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}
}
