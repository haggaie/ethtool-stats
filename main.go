package main

import (
	"fmt"
	"log"
	"os"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/safchain/ethtool"
)

func main() {
	ethHandle, err := ethtool.NewEthtool()
	if err != nil {
		panic(err.Error())
	}
	defer ethHandle.Close()

	c, err := statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}

	stat_names := []string{
		"rx_vport_unicast_packets",
		"rx_vport_unicast_bytes",
		"tx_vport_unicast_packets",
		"tx_vport_unicast_bytes",
	}

	// prefix every metric with the app name
	c.Namespace = "ethtool.stats."
	netdev := os.Args[1]
	c.Tags = append(c.Tags, fmt.Sprintf("netdev:%s", netdev))

	stats, err := ethHandle.Stats(netdev)
	if err != nil {
		panic(err.Error())
	}

	for _, stat_name := range stat_names {
		fmt.Printf("%s: %d\n", stat_name, stats[stat_name])
		err = c.Count(stat_name, int64(stats[stat_name]), nil, 1)
	}
}
