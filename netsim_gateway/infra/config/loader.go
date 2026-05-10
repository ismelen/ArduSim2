package config

import (
	"encoding/json"
	"net"
	"netsim_gateway/usecase"
	"os"
)

type Config struct {
	UAVListenPort     int                     `json:"uav_listen_port"`
	NetsimListenPort  int                     `json:"netsim_listen_port"`
	SnapshotIntervalS int                     `json:"snapshot_interval_s"`
	NetsimDiscovery   usecase.DiscoveryConfig `json:"netsim_discovery"`
	Log               LogConfig               `json:"log"`
}

type LogConfig struct {
	Level string `json:"level"`
}

func LoadConfig(path string) Config {
	file, err := os.Open(path)
	cfg := Config{
		UAVListenPort:     3000,
		NetsimListenPort:  3001,
		SnapshotIntervalS: 1,
		NetsimDiscovery: usecase.DiscoveryConfig{
			Mode:                "static",
			Addrs:               []string{"netsim_1:3000", "netsim_2:3000", "netsim_3:3000"},
			SwarmService:        "netsim",
			RediscoverIntervalS: 30,
		},
		Log: LogConfig{Level: "info"},
	}

	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		_ = decoder.Decode(&cfg)
	}

	return cfg
}

func DiscoverNetsims(cfg usecase.DiscoveryConfig) []*net.UDPAddr {
	var resolved []*net.UDPAddr

	if cfg.Mode == "static" {
		for _, addrStr := range cfg.Addrs {
			if addr, err := net.ResolveUDPAddr("udp", addrStr); err == nil {
				resolved = append(resolved, addr)
			}
		}
	} else if cfg.Mode == "swarm" {
		ips, err := net.LookupHost(cfg.SwarmService)
		if err == nil {
			for _, ip := range ips {
				addrStr := net.JoinHostPort(ip, "3000")
				if addr, err := net.ResolveUDPAddr("udp", addrStr); err == nil {
					resolved = append(resolved, addr)
				}
			}
		}
	}

	return resolved
}
