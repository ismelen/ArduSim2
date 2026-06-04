package docker

import (
	"encoding/json"
	"os"
)

// ResourceLimits holds the optional per-service Docker resource constraints.
type ResourceLimits struct {
	RAM string  // e.g. "512m", "" means no limit
	CPU float64 // e.g. 0.5, 0 means no limit
}

// HasLimits returns true if at least one limit is set.
func (rl ResourceLimits) HasLimits() bool {
	return rl.RAM != "" || rl.CPU > 0
}

// ParseResourceLimits extracts ram_limit and cpu_limit from a generic config map.
func ParseResourceLimits(cfg map[string]interface{}) ResourceLimits {
	var limits ResourceLimits

	if val, ok := cfg["ram_limit"]; ok {
		if str, ok := val.(string); ok {
			limits.RAM = str
		}
	}

	if val, ok := cfg["cpu_limit"]; ok {
		switch v := val.(type) {
		case float64:
			limits.CPU = v
		case float32:
			limits.CPU = float64(v)
		case int:
			limits.CPU = float64(v)
		case int64:
			limits.CPU = float64(v)
		}
	}

	return limits
}

// LoadRawConfig reads a JSON file into a generic map, useful for parsing
// resource limits before templating or modifications.
func LoadRawConfig(path string) map[string]interface{} {
	cfg := make(map[string]interface{})
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	return cfg
}
