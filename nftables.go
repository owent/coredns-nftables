package coredns_nftables

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"
	"github.com/miekg/dns"
	"github.com/vishvananda/netns"

	"github.com/google/nftables"
)

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
	nw := nonwriter.New(w)
	rcode, err := plugin.NextOrFailure(m.Name(), m.Next, ctx, nw, r)
	if err != nil {
		return rcode, err
	}

	r = nw.Msg
	if r == nil {
		return 1, fmt.Errorf("no answer received")
	}

	if r.Answer == nil {
		log.Debug("Request didn't contain any answer")
		return 0, nil
	}
	var hasValidRecord bool = false
	for _, answer := range r.Answer {
		if answer.Header().Rrtype == dns.TypeA || answer.Header().Rrtype == dns.TypeAAAA {
			hasValidRecord = true
			break
		}
	}
	if !hasValidRecord {
		log.Debug("Request didn't contain any A/AAAA record")
		return 0, nil
	}

	// Create a new network namespace to test these operations,
	// and tear down the namespace at test completion.
	c, newNS := openSystemNFTConn()
	defer cleanupSystemNFTConn(newNS)
	// Clear all rules at the beginning + end of the test.
	c.FlushRuleset()
	defer c.FlushRuleset()

	cache := NftablesCache{}
	for _, answer := range r.Answer {
		var tableFamilies []nftables.TableFamily
		switch answer.Header().Rrtype {
		case dns.TypeA:
			{
				recordCount.WithLabelValues(metrics.WithServer(ctx)).Inc()
				defer exportRecordDuration(ctx, time.Now())

				tableFamilies = []nftables.TableFamily{nftables.TableFamilyIPv4, nftables.TableFamilyINet, nftables.TableFamilyBridge}
			}
		case dns.TypeAAAA:
			{
				recordCount.WithLabelValues(metrics.WithServer(ctx)).Inc()
				defer exportRecordDuration(ctx, time.Now())

				tableFamilies = []nftables.TableFamily{nftables.TableFamilyIPv6, nftables.TableFamilyINet, nftables.TableFamilyBridge}
			}
		default:
			continue
		}

		for _, family := range tableFamilies {
			ruleSet, ok := m.Rules[family]
			if ok {
				for _, rule := range ruleSet.RuleAddElement {
					err := rule.ServeDNS(ctx, &cache, &answer, c, family)
					if err != nil {
						switch answer.Header().Rrtype {
						case dns.TypeA:
							log.Errorf("Add element %v(%v) to %v %v %v failed.%v", answer.(*dns.A).A.String(), answer.Header().Name, cache.GetFamilyName(family), rule.TableName, rule.SetName, err)
						case dns.TypeAAAA:
							log.Errorf("Add element %v(%v) to %v %v %v failed.%v", answer.(*dns.AAAA).AAAA.String(), answer.Header().Name, cache.GetFamilyName(family), rule.TableName, rule.SetName, err)
						default:
							log.Errorf("Add element %v(%v) to %v %v %v failed.%v", answer.String(), answer.Header().Name, cache.GetFamilyName(family), rule.TableName, rule.SetName, err)
						}
					}
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

// openSystemNFTConn returns a netlink connection that tests against
// the running kernel in a separate network namespace.
// cleanupSystemNFTConn() must be called from a defer to cleanup
// created network namespace.
func openSystemNFTConn() (*nftables.Conn, netns.NsHandle) {
	// We lock the goroutine into the current thread, as namespace operations
	// such as those invoked by `netns.New()` are thread-local. This is undone
	// in cleanupSystemNFTConn().
	runtime.LockOSThread()

	ns, err := netns.New()
	if err != nil {
		log.Fatalf("netns.New() failed: %v", err)
	}
	c, err := nftables.New(nftables.WithNetNSFd(int(ns)))
	if err != nil {
		log.Fatalf("nftables.New() failed: %v", err)
	}
	return c, ns
}

func cleanupSystemNFTConn(newNS netns.NsHandle) {
	defer runtime.UnlockOSThread()

	if err := newNS.Close(); err != nil {
		log.Fatalf("newNS.Close() failed: %v", err)
	}
}
