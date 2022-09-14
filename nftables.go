package coredns_nftables

import (
	"context"
	"fmt"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	clog "github.com/coredns/coredns/plugin/pkg/log"

	"github.com/miekg/dns"

	"github.com/google/nftables"
)

var log = clog.NewWithPlugin("nftables")

type NftablesRuleSet struct {
	RuleAddElement []*NftablesSetAddElement
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

	if r == nil {
		return 1, fmt.Errorf("no answer received")
	}

	if r.Answer == nil {
		log.Debug("Request didn't contain any answer")
		return 0, nil
	}

	for _, answer := range r.Answer {
		var tableFamilies []nftables.TableFamily
		if answer.Header().Rrtype == dns.TypeA {
			recordCount.WithLabelValues(metrics.WithServer(ctx)).Inc()
			defer exportRecordDuration(ctx, time.Now())

			tableFamilies = []nftables.TableFamily{nftables.TableFamilyIPv4, nftables.TableFamilyINet, nftables.TableFamilyBridge}
		} else if answer.Header().Rrtype == dns.TypeAAAA {
			recordCount.WithLabelValues(metrics.WithServer(ctx)).Inc()
			defer exportRecordDuration(ctx, time.Now())

			tableFamilies = []nftables.TableFamily{nftables.TableFamilyIPv6, nftables.TableFamilyINet, nftables.TableFamilyBridge}
		}

		for _, family := range tableFamilies {
			ruleSet, ok := m.Rules[family]
			if ok {
				for _, rule := range ruleSet.RuleAddElement {
					rule.ServeDNS(ctx, w, r, family)
				}
			}
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

func exportRecordDuration(ctx context.Context, start time.Time) {
	recordDuration.WithLabelValues(metrics.WithServer(ctx)).
		Observe(float64(time.Since(start).Microseconds()))
}
