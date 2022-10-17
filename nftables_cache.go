package coredns_nftables

import (
	"container/list"
	"sync"
	"time"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/google/nftables"
	"github.com/vishvananda/netns"
)

var log = clog.NewWithPlugin("nftables")
var cacheLock sync.Mutex = sync.Mutex{}
var cacheList = list.New()
var cacheExpiredDuration time.Duration = time.Minute * time.Duration(5)

type NftableCache struct {
	table    *nftables.Table
	setCache map[string]*map[string]time.Time
}

type NftablesCache struct {
	tables                    map[nftables.TableFamily]*map[string]*NftableCache
	CreateTimepoint           time.Time
	NftableConnection         *nftables.Conn
	NetworkNamespace          netns.NsHandle
	HasNftableConnectionError bool
}

func NewCache() (*NftablesCache, error) {

	{
		cacheLock.Lock()
		defer cacheLock.Unlock()

		// Destroy timeout connections
		for cacheList.Front() != nil {
			cacheHead := cacheList.Front().Value.(*NftablesCache)
			cacheList.Remove(cacheList.Front())

			if time.Since(cacheHead.CreateTimepoint) > cacheExpiredDuration {
				go cacheHead.destroy()
			} else {
				log.Debugf("nftables connection select %p from pool", cacheHead)
				return cacheHead, nil
			}
		}
	}

	c, newNS, err := openSystemNFTConn()
	if err != nil {
		return nil, err
	}

	ret := &NftablesCache{
		tables:                    make(map[nftables.TableFamily]*map[string]*NftableCache),
		CreateTimepoint:           time.Now(),
		NftableConnection:         c,
		NetworkNamespace:          newNS,
		HasNftableConnectionError: false,
	}

	log.Debugf("nftables connection create %p", ret)
	return ret, nil
}

func (cache *NftablesCache) destroy() error {
	log.Debugf("nftables connection %p start to destroy", cache)

	cleanupSystemNFTConn(cache.NetworkNamespace)
	return nil
}

func CloseCache(cache *NftablesCache) error {
	err := cache.NftableConnection.Flush()
	if err != nil {
		log.Errorf("Flush nftables connection failed %v", err)
		cache.HasNftableConnectionError = true
	}

	if cache.HasNftableConnectionError || time.Since(cache.CreateTimepoint) > cacheExpiredDuration {
		return cache.destroy()
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()

	cacheList.PushBack(cache)
	log.Debugf("nftables connection %p add to cache pool", cache)

	return nil
}

func ClearCache() {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	// Destroy timeout connections
	for cacheList.Front() != nil {
		cacheHead := cacheList.Front().Value.(*NftablesCache)
		cacheList.Remove(cacheList.Front())

		go cacheHead.destroy()
	}
}

func (cache *NftablesCache) MutableNftablesTable(family nftables.TableFamily, tableName string) *NftableCache {
	tableSet, ok := (*cache).tables[family]
	if !ok {
		tableSetM := make(map[string]*NftableCache)
		tableSet = &tableSetM
		(*cache).tables[family] = tableSet
	}

	if len(*tableSet) == 0 {
		familName := (*cache).GetFamilyName(family)
		tables, _ := cache.NftableConnection.ListTablesOfFamily(family)
		if tables != nil {
			log.Debugf("Nftable %v table(s) of %v found", len(tables), familName)
			for _, table := range tables {
				log.Debugf("\t - %v", table.Name)
				(*tableSet)[(*table).Name] = &NftableCache{
					table: table,
				}
			}
		}
	}

	tableCache, ok := (*tableSet)[tableName]
	if !ok {
		tableCache = &NftableCache{
			table: &nftables.Table{
				Family: family,
				Name:   tableName,
			},
		}
		log.Debugf("Nftable try to create table %v %v", (*cache).GetFamilyName(family), tableName)
		(*tableSet)[tableName] = tableCache
		tableCache.table = cache.NftableConnection.AddTable(tableCache.table)
	}

	return tableCache
}

func (cache *NftablesCache) SetAddElements(tableCache *NftableCache, set *nftables.Set, elements []nftables.SetElement) error {
	err := cache.NftableConnection.SetAddElements(set, elements)
	if err != nil {
		cache.HasNftableConnectionError = true
	}

	return err
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
