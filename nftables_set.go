package coredns_nftables

import (
	"context"
	"time"

	"github.com/google/nftables"
	"github.com/miekg/dns"
)

type NftablesSetAddElement struct {
	TableName string
	SetName   string
	Interval  bool
	Timeout   time.Duration
	KeyType   nftables.SetDatatype
}

func (m *NftablesSetAddElement) Name() string { return "nftables-set-add-element" }

func (m *NftablesSetAddElement) ServeDNS(ctx context.Context, cache *NftablesCache, answer *dns.RR, cc *nftables.Conn, family nftables.TableFamily) error {
	var elements []nftables.SetElement
	var element_text string
	switch (*answer).Header().Rrtype {
	case dns.TypeA:
		elements = []nftables.SetElement{{Key: (*answer).(*dns.A).A}}
		element_text = (*answer).(*dns.A).A.String()
	case dns.TypeAAAA:
		elements = []nftables.SetElement{{Key: (*answer).(*dns.AAAA).AAAA}}
		element_text = (*answer).(*dns.AAAA).AAAA.String()
	default:
		return nil
	}

	table := cache.MutableNftablesTable(cc, family, m.TableName)
	// get old set
	set, _ := cc.GetSetByName(table, m.SetName)
	if set == nil {
		// Create nftable set if KeyType is not nftables.TypeInvalid
		if m.KeyType == nftables.TypeInvalid {
			log.Debugf("Nftable set %v %v %v not found and %s is ignored", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
			return nil
		}
		portSet := &nftables.Set{
			Table:      table,
			Name:       m.SetName,
			KeyType:    m.KeyType,
			Interval:   m.Interval,
			HasTimeout: m.Timeout.Microseconds() > 0,
			Timeout:    m.Timeout,
		}

		log.Debugf("Nftable create set %v %v %v and add element %s", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
		return cc.AddSet(portSet, elements)
	}

	// Insert into set
	log.Debugf("Nftable set %v %v %v add element %s", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
	return cc.SetAddElements(set, elements)
}
