package config

import (
	"encoding/json"
	"netsim/usecase"
	"os"
)

// Config is a flat struct that maps directly to the flat config.json.
type Config struct {
	ListenPort        int     `json:"listen_port"`
	GatewayAddr       string  `json:"gateway_addr"`
	LossMode          string  `json:"loss_mode"`
	BufferSizeBytes   int     `json:"buffer_size_bytes"`
	CsmaRangeM        float64 `json:"csma_range_m"`
	MaxCsmaRetries    uint32  `json:"max_csma_retries"`
	ChunkSizeM        float64 `json:"chunk_size_m"`
	ChunkRadius       int64   `json:"chunk_radius"`
	MaxRangeM         float64 `json:"max_range_m"`
	SnapshotIntervalS int     `json:"snapshot_interval_s"`
	Level             string  `json:"level"`
	LoggerAddr        string  `json:"logger_addr"`
}

// SimulationConfig extracts the usecase.Config from the flat Config.
func (c Config) SimulationConfig() usecase.Config {
	return usecase.Config{
		LossMode:          c.LossMode,
		BufferSizeBytes:   c.BufferSizeBytes,
		CsmaRangeM:        c.CsmaRangeM,
		MaxCsmaRetries:    c.MaxCsmaRetries,
		ChunkSizeM:        c.ChunkSizeM,
		ChunkRadius:       c.ChunkRadius,
		MaxRangeM:         c.MaxRangeM,
		SnapshotIntervalS: c.SnapshotIntervalS,
	}
}

func LoadConfig(path string) Config {
	file, err := os.Open(path)
	cfg := Config{
		ListenPort:        3000,
		GatewayAddr:       "netsim_gateway:3001",
		LossMode:          "realistic",
		BufferSizeBytes:   163840,
		CsmaRangeM:        700.0,
		MaxCsmaRetries:    5,
		ChunkSizeM:        350.0,
		ChunkRadius:       2,
		MaxRangeM:         1350.0,
		SnapshotIntervalS: 1,
		Level:             "info",
		LoggerAddr:        "logger:5000",
	}

	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		_ = decoder.Decode(&cfg)
	}

	return cfg
}
