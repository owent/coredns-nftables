# coredns-nftables

nftables plugin of coredns

## Name

*nftables* - Modify nftables after got a DNS response message.

## Compilation

```txt
nftables:github.com/owent/coredns-nftables
```

This plugin should be add after [finalize](https://coredns.io/explugins/finalize/).

```bash
echo "finalize:github.com/tmeckel/coredns-finalizer
nftables:github.com/owent/coredns-nftables
" >> plugin.cfg
```

## Syntax

```corefile
nftables [ip/ip6/inet/bridge]... {
  set add element <TABLE_NAME> <SET_NAME> [ip/ip6/inet] [interval] [timeout]
}
```

Valid timeout units are "ms", "s", "m", "h".

## Examples

Enable nftables:

```corefile
example.org {
    whoami
    forward . 8.8.8.8
    finalize
    nftables ip ip6 inet bridge {
      set add element filter IPSET inet false 24h
    }
}
```

## See Also
