package coredns_nftables

import (
	"context"

	"github.com/miekg/dns"
)

type NftablesSetAddElement struct {
	TableName string
	SetName   string
}

func (m *NftablesSetAddElement) Name() string { return "nftables-set-add-element" }

func (m *NftablesSetAddElement) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return 0, nil
}
