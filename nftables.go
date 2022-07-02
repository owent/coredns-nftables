package coredns_nftables

import (
	"context"

	"github.com/coredns/coredns/plugin"

	"github.com/miekg/dns"

	"github.com/google/nftables"
)

type NftablesRuleSet struct {
	Rule []plugin.Handler
}

// NftablesHandler implements the plugin.Handler interface.
type NftablesHandler struct {
	Next plugin.Handler

	Rules map[nftables.TableFamily]*NftablesRuleSet
}

func (m *NftablesHandler) Name() string { return "nftables" }

func (m *NftablesHandler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	rcode, err := plugin.NextOrFailure(m.Name(), m.Next, ctx, w, r)
	if err != nil {
		return rcode, err
	}

	ret, ok := m.Rules[nftables.TableFamilyIPv4]
	if ok {
		for _, rule := range ret.Rule {
			rule.ServeDNS(ctx, w, r)
		}
	}

	return 0, nil
}

func (m *NftablesHandler) MutableRuleSet(family nftables.TableFamily) *NftablesRuleSet {
	ret, ok := m.Rules[family]
	if ok {
		return ret
	} else {
		ret = &NftablesRuleSet{}
		m.Rules[family] = ret
		return ret
	}
}
