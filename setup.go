package coredns_nftables

import (
	"strconv"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/google/nftables"
)

func init() {
	plugin.Register("nftables", setup)
}

func setup(c *caddy.Controller) error {
	handle := NewNftablesHandler()
	err := parse(c, &handle)
	if err != nil {
		return plugin.Error("nftables", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		handle.Next = next
		ClearCache()
		return &handle
	})

	log.Debug("Add nftables plugin to dnsserver")

	return nil
}

func parse(c *caddy.Controller, handle *NftablesHandler) error {
	for c.Next() {
		var families []nftables.TableFamily
		// nftables [family...]
		args := c.RemainingArgs()
		allowAutoIpAddr := true
		if len(args) > 0 {
			for _, family := range args {
				switch strings.ToLower(family) {
				case "ip":
					families = append(families, nftables.TableFamilyIPv4)
				case "ip6":
					families = append(families, nftables.TableFamilyIPv6)
				case "inet":
					families = append(families, nftables.TableFamilyINet)
					allowAutoIpAddr = false
				case "arp":
					families = append(families, nftables.TableFamilyARP)
					allowAutoIpAddr = false
				case "bridge":
					families = append(families, nftables.TableFamilyBridge)
					allowAutoIpAddr = false
				case "netdev":
					families = append(families, nftables.TableFamilyNetdev)
					allowAutoIpAddr = false
				}
			}
		}
		// inet support both ipv4 and ipv6
		if len(families) == 0 {
			families = append(families, nftables.TableFamilyINet)
		}

		// Refinements? In an extra block.
		for c.NextBlock() {
			switch strings.ToLower(c.Val()) {
			// first number is cap, second is an new ttl
			case "set":
				{
					args := c.RemainingArgs()
					if len(args) < 1 {
						return c.Errf("nftables set argument count invalid")
					}
					var err error = nil
					if strings.ToLower(args[0]) == "add" {
						err = setupSetAddElement(c, handle, allowAutoIpAddr, families, args)
					} else if strings.ToLower(args[0]) == "lru" {
						err = setupSetLruOptions(c, handle, args)
					} else {
						return c.Errf("nftables set action %v invalid", args[0])
					}
					if err != nil {
						return err
					}
				}

			case "connection":
				{
					args := c.RemainingArgs()
					if len(args) < 2 {
						return c.Errf("nftables set argument count invalid")
					}
					connectionAction := strings.ToLower(args[0])
					if connectionAction != "timeout" {
						return c.Errf("nftables connection action %v invalid", connectionAction)
					}

					parseTimeout, err := time.ParseDuration(args[1])
					if err != nil {
						return c.Errf("nftables connection action %v argument %v invalid, %v", connectionAction, args[1], err)
					}
					SetConnectionTimeout(parseTimeout)
				}

			case "async":
				{
					args := c.RemainingArgs()
					if len(args) < 1 {
						return c.Errf("nftables set argument count invalid")
					}

					parseAsync, err := strconv.ParseBool(args[0])
					if err != nil {
						return c.Errf("nftables async argument %v invalid, %v", args[0], err)
					}

					SetNftableAsyncMode(parseAsync)
				}

			default:
				return c.ArgErr()
			}
		}

		log.Debug("Successfully parsed configuration")
	}

	return nil
}

func setupSetAddElement(c *caddy.Controller, handle *NftablesHandler, allowAutoIpAddr bool, families []nftables.TableFamily, args []string) error {
	if len(args) <= 3 {
		return c.Errf("nftables set add element argument count invalid")
	}

	setRuleAction := strings.ToLower(args[0])
	setRuleTarget := strings.ToLower(args[1])
	setRuleTableName := args[2]
	setRuleSetName := args[3]
	var setRuleIsInterval bool = false
	var setRuleTimeout time.Duration = 0 // time.ParseDuration()
	var keyType nftables.SetDatatype = nftables.TypeInvalid
	if setRuleAction != "add" || setRuleTarget != "element" {
		return c.Errf("nftables set action %v invalid", setRuleTarget)
	}
	var nextArgIndex int = 4

	if len(args) > nextArgIndex {
		tryKeyType := strings.ToLower(args[nextArgIndex])
		if tryKeyType == "ip" {
			keyType = nftables.TypeIPAddr
			nextArgIndex += 1
		} else if tryKeyType == "ip6" {
			keyType = nftables.TypeIP6Addr
			nextArgIndex += 1
		} else if tryKeyType == "auto" {
			keyType = nftables.TypeInvalid // Use invalid as auto
			nextArgIndex += 1
		}
	}
	if keyType == nftables.TypeInvalid && !allowAutoIpAddr {
		return c.Errf("nftables set action %v address type invalid, only ip and ip6 family support auto address type", setRuleTarget)
	}

	if len(args) > nextArgIndex {
		tryInterval := strings.ToLower(args[nextArgIndex])
		if parseBool, err := strconv.ParseBool(tryInterval); err == nil {
			setRuleIsInterval = parseBool
			nextArgIndex += 1
		}
	}

	if len(args) > nextArgIndex {
		if parseTimeout, err := time.ParseDuration(args[nextArgIndex]); err == nil {
			setRuleTimeout = parseTimeout
			nextArgIndex += 1
		}
	}

	for i := nextArgIndex; i < len(args); i++ {
		log.Warningf("Ignore invalid setting %s", args[i])
	}

	rule := NftablesSetAddElement{TableName: setRuleTableName, SetName: setRuleSetName, Interval: setRuleIsInterval, Timeout: setRuleTimeout, KeyType: keyType}

	for _, family := range families {
		ruleSet := handle.MutableRuleSet(family)
		ruleSet.RuleAddElement = append(ruleSet.RuleAddElement, &rule)
	}

	return nil
}

func setupSetLruOptions(c *caddy.Controller, handle *NftablesHandler, args []string) error {
	if len(args) <= 2 {
		return c.Errf("nftables set lru argument count invalid")
	}

	if strings.ToLower(args[1]) == "max" {
		parseRetryTimes, err := strconv.ParseInt(args[2], 10, 32)
		if err != nil {
			return c.Errf("nftables set lru max, can not convert %v to integer, %v", args[2], err)
		}

		SetSetLruMaxCount(int(parseRetryTimes))
	} else if strings.ToLower(args[1]) == "timeout" {
		parseTimeout, err := time.ParseDuration(args[2])
		if err != nil {
			return c.Errf("nftables set lru timeout argument %v invalid, %v", args[2], err)
		}
		SetSetLruTimeout(parseTimeout)
	} else if strings.ToLower(args[1]) == "retry" {
		if len(args) <= 3 {
			return c.Errf("nftables set lru retry argument count invalid")
		}
		if strings.ToLower(args[2]) != "times" {
			return c.Errf("nftables set lru retry %v unknown option", args[2])
		}

		parseRetryTimes, err := strconv.ParseInt(args[3], 10, 32)
		if err != nil {
			return c.Errf("nftables set lru retry %v can not convert %v to integer, %v", args[2], args[3], err)
		}

		SetSetLruMaxRetryTimes(int(parseRetryTimes))
	} else {
		return c.Errf("nftables set lru %v unknown option", args[1])
	}

	return nil
}
