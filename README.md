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
  set add element <TABLE_NAME> <SET_NAME> [ip/ip6/inet/auto] [interval] [timeout]
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

## For Developers

### Debug Build

```bash
git clone --depth 1 https://github.com/coredns/coredns.git coredns
cd coredns
git reset --hard
echo "finalize:github.com/tmeckel/coredns-finalizer" >> plugin.cfg
echo "nftables:github.com/owent/coredns-nftables" >> plugin.cfg
go get github.com/tmeckel/coredns-finalizer
go get github.com/owent/coredns-nftables
go generate

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/linux/amd64/coredns
```

### Configure File For Debug

```conf

(default_dns_ip) {
  errors
  forward . 119.29.29.29 223.5.5.5 1.0.0.1 94.140.14.140 2402:4e00:: 2400:3200::1 2400:3200:baba::1 2606:4700:4700::1001 2a10:50c0::1:ff {
    policy sequential
  }
  loop
  log
}

. {
  import default_dns_ip
}

owent.net www.owent.net {
  finalize
  nftables ip ip6 bridge {
    set add element test_coredns_nft TEST_SET auto false 24h
  }
  import default_dns_ip
}

```

### Run

`build/linux/amd64/coredns -dns.port=6813 -conf test-coredns.conf`
