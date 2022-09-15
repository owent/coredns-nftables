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
	handle := NftablesHandler{Next: nil, Rules: make(map[nftables.TableFamily]*NftablesRuleSet)}
	err := parse(c, &handle)
	if err != nil {
		return plugin.Error("nftables", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		handle.Next = next
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
		if len(args) > 0 {
			for _, family := range args {
				switch strings.ToLower(family) {
				case "ip":
					families = append(families, nftables.TableFamilyIPv4)
				case "ip6":
					families = append(families, nftables.TableFamilyIPv6)
				case "inet":
					families = append(families, nftables.TableFamilyINet)
				case "arp":
					families = append(families, nftables.TableFamilyARP)
				case "bridge":
					families = append(families, nftables.TableFamilyBridge)
				case "netdev":
					families = append(families, nftables.TableFamilyNetdev)
				}
			}
		}
		// Just like nftables,
		if len(families) == 0 {
			families = append(families, nftables.TableFamilyIPv4)
		}

		// Refinements? In an extra block.
		for c.NextBlock() {
			switch strings.ToLower(c.Val()) {
			// first number is cap, second is an new ttl
			case "set":
				args := c.RemainingArgs()
				if len(args) <= 3 {
					return c.ArgErr()
				}
				setRuleAction := strings.ToLower(args[0])
				setRuleTarget := strings.ToLower(args[1])
				setRuleTableName := args[2]
				setRuleSetName := args[3]
				var setRuleIsInterval bool = false
				var setRuleTimeout time.Duration = 0 // time.ParseDuration()
				var keyType nftables.SetDatatype = nftables.TypeInetService
				if setRuleAction != "add" || setRuleTarget != "element" {
					return c.ArgErr()
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
					} else if tryKeyType == "inet" {
						keyType = nftables.TypeInetService
						nextArgIndex += 1
					}
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

			default:
				return c.ArgErr()
			}
		}

		log.Debug("Successfully parsed configuration")
	}

	return nil
}
