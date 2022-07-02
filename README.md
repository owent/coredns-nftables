# coredns-nftables
nftables plugin of coredns

## Name

*nftables* - Modify nftables after got a DNS response message.

## Description

TODO

## Syntax

~~~ txt
nftables:github.com/owent/coredns-nftables
~~~

## Examples

Enable nftables:

~~~ corefile
example.org {
    whoami
    forward . 8.8.8.8
    nftables ip ip6 inet bridge {
      set add element TABLE_NAME IPSET
    }
}
~~~

## See Also

