package config

import (
	"encoding/json"
	"netsim/usecase"
	"os"
)

type Config struct {
	NodeID       string         `json:"node_id"`
	ListenPort   int            `json:"listen_port"`
	GatewayAddr  string         `json:"gateway_addr"`
	Simulation   usecase.Config `json:"simulation"`
	Log          LogConfig      `json:"log"`
}

type LogConfig struct {
	Level string `json:"level"`
}

func LoadConfig(path string) Config {
	file, err := os.Open(path)
	cfg := Config{
		NodeID:      "netsim_1",
		ListenPort:  3000,
		GatewayAddr: "gateway:3001",
		Simulation: usecase.Config{
			BufferSizeBytes:   163840,
			CsmaRangeM:        700.0,
			MaxCsmaRetries:    5,
			ChunkSizeM:        350.0,
			ChunkRadius:       2,
			MaxRangeM:         1350.0,
			SnapshotIntervalS: 1,
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
