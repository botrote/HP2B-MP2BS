# MP2BS

## MP2 Blockchain Session Protocol

To test MP2BS.. 

```sh
cd test/mp2bsTest/
```

There are two example folders. Each folder has a README file that includes how to test mp2bs.  

1. one-to-one test 
2. push&pull test 

## Multipath setup

The routing table must be set up to use a specific source IP before testing.

```sh
sudo ip rule add from [ip address] table [number]
sudo ip route add [network address]/[subnet mask] dev [NIC name] scope link table [number]
sudo ip route add default via [gateway address] dev [NIC name] table [number]
```

### Example 

<img width="80%" src="https://github.com/MCNL-HGU/mp2bs/assets/52683010/bf0c5529-ff59-4ac9-bed1-18cba6e97bba"/>

```sh
sudo ip rule add from 192.168.10.25 table 1
sudo ip route add 192.168.10.0/24 dev eth0 scope link table 1
sudo ip route add default via 192.168.10.1 dev eth0 table 1

sudo ip rule add from 192.168.10.26 table 2
sudo ip route add 192.168.10.0/24 dev eth1 scope link table 2
sudo ip route add default via 192.168.10.1 dev eth1 table 2
```