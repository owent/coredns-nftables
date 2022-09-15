package coredns_nftables

import (
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/google/nftables"
)

var log = clog.NewWithPlugin("nftables")

type NftablesCache struct {
	tables map[nftables.TableFamily]*map[string]*nftables.Table
}

func (cache *NftablesCache) MutableNftablesTable(cc *nftables.Conn, family nftables.TableFamily, tableName string) *nftables.Table {
	tableSet, ok := (*cache).tables[family]
	if !ok {
		tableSetM := make(map[string]*nftables.Table)
		tableSet = &tableSetM
		(*cache).tables[family] = tableSet
	}

	if len(*tableSet) == 0 {
		familName := (*cache).GetFamilyName(family)
		tables, _ := cc.ListTablesOfFamily(family)
		if tables != nil {
			for _, table := range tables {
				log.Debugf("Nftable table %v %v found", familName, tableName)
				(*tableSet)[(*table).Name] = table
			}
		}
	}

	table, ok := (*tableSet)[tableName]
	if !ok {
		table = &nftables.Table{
			Family: family,
			Name:   tableName,
		}
		log.Debugf("Nftable try to create table %v %v", (*cache).GetFamilyName(family), tableName)
		table = cc.AddTable(table)
	}

	return table
}

func (cache *NftablesCache) GetFamilyName(family nftables.TableFamily) string {
	switch family {
	case nftables.TableFamilyUnspecified:
		return "unspecified"
	case nftables.TableFamilyINet:
		return "inet"
	case nftables.TableFamilyIPv4:
		return "ipv4"
	case nftables.TableFamilyIPv6:
		return "ipv6"
	case nftables.TableFamilyARP:
		return "arp"
	case nftables.TableFamilyNetdev:
		return "netdev"
	case nftables.TableFamilyBridge:
		return "bridge"
	}

	return "unknown"
}
