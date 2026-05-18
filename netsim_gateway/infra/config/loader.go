package config

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"
)

type Config struct {
	UAVListenPort     int      `json:"uav_listen_port"`
	NetsimListenPort  int      `json:"netsim_listen_port"`
	SnapshotIntervalS int      `json:"snapshot_interval_s"`
	Addrs             []string 
}

func LoadConfig(path string) Config {
	file, err := os.Open(path)
	addrsStr := os.Getenv("ADDRS")
	cfg := Config{
		UAVListenPort:     3000,
		NetsimListenPort:  3001,
		SnapshotIntervalS: 1,
		Addrs:             strings.Split(addrsStr, ","),
	}

	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		_ = decoder.Decode(&cfg)
	}

	return cfg
}

func DiscoverNetsims(addrs []string) []*net.UDPAddr {
	var resolved []*net.UDPAddr
	for _, addrStr := range addrs {
		if addr, err := net.ResolveUDPAddr("udp", addrStr); err == nil {
			resolved = append(resolved, addr)
		} else {
			log.Printf("Failed to resolve %s: %v\n", addrStr, err)
		}
	}
	return resolved
}
