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

func (m *NftablesSetAddElement) ServeDNS(ctx context.Context, cache *NftablesCache, answer *dns.RR, family nftables.TableFamily) error {
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

	table := cache.MutableNftablesTable(family, m.TableName)
	// get old set
	set, _ := cache.NftableConnection.GetSetByName(table, m.SetName)
	if set == nil {
		// Create nftable set if KeyType is not nftables.TypeInvalid
		var keyType = m.KeyType
		if keyType == nftables.TypeInvalid {
			if family == nftables.TableFamilyIPv4 {
				keyType = nftables.TypeIPAddr
			} else if family == nftables.TableFamilyIPv6 {
				keyType = nftables.TypeIP6Addr
			} else {
				log.Debugf("Nftable set %v %v %v ignore element %s because set not found", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
				return nil
			}
		}

		// Ignore unmatched set
		if (*answer).Header().Rrtype == dns.TypeA && keyType == nftables.TypeIP6Addr {
			log.Debugf("Nftable set %v %v %v ignore element %s because it's a ipv6 set", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
			return nil
		} else if (*answer).Header().Rrtype == dns.TypeAAAA && keyType == nftables.TypeIPAddr {
			log.Debugf("Nftable set %v %v %v ignore element %s because it's a ipv4 set", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
			return nil
		}

		portSet := &nftables.Set{
			Table:      table,
			Name:       m.SetName,
			KeyType:    keyType,
			Interval:   m.Interval,
			HasTimeout: m.Timeout.Microseconds() > 0,
			Timeout:    m.Timeout,
		}

		log.Debugf("Nftable create set %v %v %v and add element %s", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
		err := cache.NftableConnection.AddSet(portSet, elements)
		if err != nil {
			log.Errorf("Nftable create set %v %v %v and add element %s but AddSet failed. %v", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text, err)
			return err
		}
		err = cache.NftableConnection.Flush()
		if err != nil {
			log.Errorf("Nftable create set %v %v %v and add element %s but Flush failed. %v", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text, err)
			cache.HasNftableConnectionError = true
		}
		return err
	}

	// Ignore unmatched set
	if (*answer).Header().Rrtype == dns.TypeA && set.KeyType == nftables.TypeIP6Addr {
		log.Debugf("Nftable set %v %v %v ignore element %s because it's a ipv6 set", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
		return nil
	} else if (*answer).Header().Rrtype == dns.TypeAAAA && set.KeyType == nftables.TypeIPAddr {
		log.Debugf("Nftable set %v %v %v ignore element %s because it's a ipv4 set", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
		return nil
	}
	log.Debugf("Nftable set %v %v %v add element %s", (*cache).GetFamilyName(family), m.TableName, m.SetName, element_text)
	return cache.NftableConnection.SetAddElements(set, elements)
}
