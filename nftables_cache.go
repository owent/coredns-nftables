package coredns_nftables

import (
	"sync"
	"time"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/google/nftables"
	"github.com/vishvananda/netns"
)

var log = clog.NewWithPlugin("nftables")
var cacheLock sync.Mutex = sync.Mutex{}
var cacheHead *NftablesCache = nil
var cacheExpiredDuration time.Duration = time.Minute * time.Duration(5)

type NftablesCache struct {
	prev              *NftablesCache
	next              *NftablesCache
	tables            map[nftables.TableFamily]*map[string]*nftables.Table
	CreateTimepoint   time.Time
	NftableConnection *nftables.Conn
	NetworkNamespace  netns.NsHandle
}

func NewCache() (*NftablesCache, error) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	// Destroy timeout connections
	for cacheHead != nil {
		if time.Since(cacheHead.CreateTimepoint) > cacheExpiredDuration {
			cacheNext := cacheHead.next
			if cacheNext != nil {
				cacheNext.prev = nil
			}
			cacheCurrent := cacheHead
			cacheHead = cacheNext

			go cacheCurrent.destroy()
		} else {
			cacheNext := cacheHead.next
			cacheCurrent := cacheHead
			cacheHead = cacheNext

			if cacheNext != nil {
				cacheNext.prev = nil
			}
			cacheCurrent.next = nil

			log.Debugf("nftables connection select %p from pool", cacheCurrent)

			return cacheCurrent, nil
		}
	}

	c, newNS, err := openSystemNFTConn()
	if err != nil {
		return nil, err
	}

	ret := &NftablesCache{
		prev:              nil,
		next:              nil,
		tables:            make(map[nftables.TableFamily]*map[string]*nftables.Table),
		CreateTimepoint:   time.Now(),
		NftableConnection: c,
		NetworkNamespace:  newNS,
	}

	log.Debugf("nftables connection create %p", ret)
	return ret, nil
}

func (cache *NftablesCache) destroy() error {
	log.Debugf("nftables connection %p start to destroy", cache)
	err := cache.NftableConnection.Flush()
	if err != nil {
		log.Errorf("Flush nftables connection failed %v", err)
	}

	cleanupSystemNFTConn(cache.NetworkNamespace)
	return err
}

func CloseCache(cache *NftablesCache) error {
	if time.Since(cache.CreateTimepoint) > cacheExpiredDuration {
		return cache.destroy()
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cacheHead != nil {
		tailCache := cacheHead
		for tailCache.next != nil {
			tailCache = tailCache.next
		}
		tailCache.next = cache
		cache.prev = tailCache
	} else {
		cacheHead = cache
	}
	log.Debugf("nftables connection %p add to cache pool", cache)

	return nil
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
			log.Debugf("Nftable %v table(s) of %v found", len(tables), familName)
			for _, table := range tables {
				log.Debugf("\t - %v", table.Name)
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
		(*tableSet)[tableName] = table
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

// openSystemNFTConn returns a netlink connection that tests against
// the running kernel in a separate network namespace.
// cleanupSystemNFTConn() must be called from a defer to cleanup
// created network namespace.
func openSystemNFTConn() (*nftables.Conn, netns.NsHandle, error) {
	// We lock the goroutine into the current thread, as namespace operations
	// such as those invoked by `netns.New()` are thread-local. This is undone
	// in cleanupSystemNFTConn().
	// runtime.LockOSThread()

	// ns, err := netns.New()
	// if err != nil {
	// 	log.Errorf("netns.New() failed: %v", err)
	// 	return nil, 0, err
	// }
	// c, err := nftables.New(nftables.WithNetNSFd(int(ns)))
	c, err := nftables.New()
	if err != nil {
		log.Errorf("nftables.New() failed: %v", err)
	}
	// return c, ns, err
	return c, 0, err
}

func cleanupSystemNFTConn(newNS netns.NsHandle) {
	// defer runtime.UnlockOSThread()

	if newNS == 0 {
		return
	}
	if err := newNS.Close(); err != nil {
		log.Errorf("newNS.Close() failed: %v", err)
	}
}

func SetConnectionTimeout(timeout time.Duration) {
	cacheExpiredDuration = timeout
}
