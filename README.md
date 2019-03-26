# ethtool-stats
Collect real-time netdev counters and publish using Datadog agent.

## Installation

First, set up datadog agent. Then, install ethtool-stats:

```
go get github.com/haggaie/ethtool-stats/ethtool-stats
```

## Running

```
ethtool-stats <ifname>
```
