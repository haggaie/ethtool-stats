package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
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
	num_measurements := flag.Uint("measurements", 0, "number of measurements to collect")
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

	measurements := make([][]uint64, *num_measurements)
	for i := range measurements {
		measurements[i] = make([]uint64, len(stat_names) + 1)
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

	var cur_measurement uint = 0

	for now := range ticker.C {
		now_ns := now.UnixNano()
		if *num_measurements > 0 {
			measurements[cur_measurement][0] = uint64(now_ns)
		}

		if *verbose {
			fmt.Printf("%9d ns, %9d ns since last\n",
				   now_ns,
				   now.Sub(prev_time).Nanoseconds())
		}
		stats, err := ethHandle.Stats(*netdev)
		if err != nil {
			panic(err.Error())
		}

		for i, stat_name := range stat_names {
			if *verbose {
				fmt.Printf("%s: %d\n", stat_name, stats[stat_name]-prev[stat_name])
			}
			if *enable_statsd {
				err = c.Count(stat_name, int64(stats[stat_name])-int64(prev[stat_name]), nil, 1)
			}
			if *num_measurements > 0 {
				measurements[cur_measurement][i + 1] = stats[stat_name]
			}
		}

		prev = stats
		prev_time = now
		cur_measurement += 1
		if cur_measurement >= *num_measurements && *num_measurements > 0 {
			break
		}
	}

	if *num_measurements > 0 {
		headers := append([]string{"time"}, []string(stat_names)...)

		file, err := os.Create("result.csv")
		if err != nil {
			log.Fatal("Cannot create file", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		writer.Write(headers)
		for _, row := range measurements {
			row_str := make([]string, len(headers))
			for col, val := range row {
				row_str[col] = fmt.Sprintf("%d", val)
			}
			err := writer.Write(row_str)
			if err != nil {
				log.Fatal("Cannot write row", err)
			}
		}
	}
}
