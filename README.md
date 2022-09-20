# coredns-nftables

nftables plugin of coredns

## Name

*nftables* - Modify nftables after got a DNS response message.

## Compilation

```txt
nftables:github.com/owent/coredns-nftables
```

This plugin should be add after [cache][1] and before [finalize](https://coredns.io/explugins/finalize/).

```bash
sed -i.bak -r '/finalize:.*/d' plugin.cfg
sed -i.bak '/cache:.*/a finalize:github.com/tmeckel/coredns-finalizer' plugin.cfg
go get github.com/tmeckel/coredns-finalizer

sed -i.bak -r '/nftables:.*/d' plugin.cfg
sed -i.bak '/cache:.*/i nftables:github.com/owent/coredns-nftables' plugin.cfg
go get github.com/owent/coredns-nftables

go generate
```

## Syntax

```corefile
nftables [ip/ip6]... {
  set add element <TABLE_NAME> <SET_NAME> [ip/ip6/auto] [interval] [timeout]
  [connection timeout <timeout>]
}

nftables [inet/bridge/arp/netdev]... {
  set add element <TABLE_NAME> <SET_NAME> <ip/ip6> [interval] [timeout]
  [connection timeout <timeout>]
}
```

The `timeout` should be greater than [cache][1].

Valid timeout units are "ms", "s", "m", "h".

If more than one `connection timeout <timeout>` are set, we use the last one.

## Examples

Enable nftables:

```corefile
example.org {
    whoami
    forward . 8.8.8.8
    finalize
    nftables ip ip6 {
      set add element filter IPSET auto false 24h
      connection timeout 10m
    }

    nftables inet bridge {
      set add element filter IPV4 ip false 24h
      set add element filter IPV6 ip6 false 24h
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
sed -i.bak -r '/finalize:.*/d' plugin.cfg
sed -i.bak '/cache:.*/a finalize:github.com/tmeckel/coredns-finalizer' plugin.cfg
go get github.com/tmeckel/coredns-finalizer
sed -i.bak -r '/nftables:.*/d' plugin.cfg
sed -i.bak '/cache:.*/a nftables:github.com/owent/coredns-nftables' plugin.cfg
go get -u github.com/owent/coredns-nftables@main
# go get github.com/owent/coredns-nftables@latest
go generate

env CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -gcflags=all="-N -l" -o build/linux/amd64/coredns
```

### Configure File For Debug

```conf
(default_dns_ip) {
  debug
  # errors
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
  nftables ip ip6 {
    set add element test_coredns_nft TEST_SET auto false 24h
    connection timeout 10m
  }
  nftables bridge {
    set add element test_coredns_nft TEST_SET_IPV4 ip false 24h
    set add element test_coredns_nft TEST_SET_IPV6 ip6 false 24h
  }
  import default_dns_ip
}
```

### VSCode lanch example

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Package",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}",
      "args": [
        "-dns.port=6813",
        "-conf=${workspaceFolder}/.vscode/test-coredns.conf",
        "-alsologtostderr"
      ],
      "showLog": true
    },
    {
      "name": "Launch Executable",
      "type": "go",
      "request": "launch",
      "mode": "exec",
      "program": "${workspaceFolder}/build/linux/amd64/coredns",
      "args": [
        "-dns.port=6813",
        "-conf=${workspaceFolder}/.vscode/test-coredns.conf",
        "-alsologtostderr"
      ],
      "cwd": "${workspaceFolder}/build",
      "showLog": true
    }
  ]
}
```

### Run

```bash
go get -v github.com/go-delve/delve/cmd/dlv

sudo build/linux/amd64/coredns -dns.port=6813 -conf test-coredns.conf

dig owent.net @127.0.0.1 -p 6813
```

[1]: https://coredns.io/plugins/cache/
