package coredns_nftables

import (
	"context"
	"time"

	"github.com/miekg/dns"
	"github.com/google/nftables"
)

type NftablesSetAddElement struct {
	TableName string
	SetName   string
	Interval  bool
	Timeout   time.Duration
}

func (m *NftablesSetAddElement) Name() string { return "nftables-set-add-element" }

func (m *NftablesSetAddElement) ServeDNS(_ctx context.Context, w dns.ResponseWriter, r *dns.Msg, nftables.TableFamily) (int, error) {
	// TODO get old value
	// TODO Check exists
	// TODO Create nftable set
	// TODO Insert into set
	return 0, nil
}
