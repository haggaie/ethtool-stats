package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/safchain/ethtool"
)

type statsList []string

func (i *statsList) String() string {
    return strings.Join(*i, ", ")
}

func (i *statsList) Set(value string) error {
    *i = append(*i, value)
    return nil
}

func main() {
	netdev := flag.String("ifname", "", "net device to monitor")
	interval := flag.Float64("interval", 1, "time between measurements")
	verbose := flag.Bool("v", false, "detailed output")
	enable_statsd := flag.Bool("statsd", false, "enable statsd reporting")
	usage := flag.Bool("h", false, "usage information")

	var stat_names statsList
        flag.Var(&stat_names, "stat", "ethtool stat name to measure")

	flag.Parse()

	if *netdev == "" || *usage {
		fmt.Printf("-ifname is required\n")
		*usage = true
	}

	if *usage {
		flag.PrintDefaults()
		return
	}

        if len(stat_names) == 0 {
		stat_names = []string{
			"rx_vport_unicast_packets",
			"rx_vport_unicast_bytes",
			"tx_vport_unicast_packets",
			"tx_vport_unicast_bytes",
		}
        }

	ethHandle, err := ethtool.NewEthtool()
	if err != nil {
		panic(err.Error())
	}
	defer ethHandle.Close()

        var c *statsd.Client
        if *enable_statsd {
		c, err := statsd.New("127.0.0.1:8125")
		if err != nil {
			log.Fatal(err)
		}

		// prefix every metric with the app name
		c.Namespace = "ethtool.stats."
		c.Tags = append(c.Tags, fmt.Sprintf("netdev:%s", *netdev))
	}

	ticker := time.NewTicker(time.Duration(*interval * 1e9) * time.Nanosecond)

	prev_time := time.Now()
	prev, err := ethHandle.Stats(*netdev)
	if err != nil {
		panic(err.Error())
	}

	start := time.Now()

	for now := range ticker.C {
		if *verbose {
			fmt.Printf("%9d ns, %9d ns since last\n",
				   now.Sub(start).Nanoseconds(),
				   now.Sub(prev_time).Nanoseconds())
		}
		stats, err := ethHandle.Stats(*netdev)
		if err != nil {
			panic(err.Error())
		}

		for _, stat_name := range stat_names {
			if *verbose {
				fmt.Printf("%s: %d\n", stat_name, stats[stat_name]-prev[stat_name])
			}
			if *enable_statsd {
				err = c.Count(stat_name, int64(stats[stat_name])-int64(prev[stat_name]), nil, 1)
			}
		}

		prev = stats
		prev_time = now
	}
}
