package domain

import (
	"regexp"
	"strings"
	"time"
)

// GeneralConfig contains simulation-wide parameters like wind and battery.
type GeneralConfig struct {
	DefaultUAVSpeed          float64 `json:"defaultUAVSpeed"`
	DefaultArduPilotInstance string  `json:"defaultArduPilotInstance"`
	LoggingEnabled           bool    `json:"loggingEnabled"`
	BatteryRestricted        bool    `json:"batteryRestricted"`
	BatteryCapacity          int     `json:"batteryCapacity"`
	VerboseLogging    bool    `json:"verboseLogging"`
	StoreLocalData    bool    `json:"storeLocalData"`
	WindEnabled       bool    `json:"windEnabled"`
	WindDirection     float64 `json:"windDirection"`
	WindSpeed         float64 `json:"windSpeed"`

	SimulationName         string `json:"simulationName"`
	OriginalSimulationName string `json:"originalSimulationName"`

	// Ground Formation configuration
	GroundFormation     string  `json:"groundFormation"` // LINEAR, MATRIX, CIRCLE, RANDOM
	FormationCenterLat  float64 `json:"formationCenterLat"`
	FormationCenterLon  float64 `json:"formationCenterLon"`
	FormationSpacing    float64 `json:"formationSpacing"`
	FormationCenterMode string  `json:"formationCenterMode"`

	// Remote Docker API endpoint for Swarm deployments (format: "IP:PORT").
	SwarmHost string `json:"swarmHost"`

	// NetsimInstances is the number of netsim worker containers to deploy.
	NetsimInstances int `json:"netsimInstances"`

	// NetsimMode sets the loss_mode for netsim instances ("realistic" or "fixed_range").
	// Empty string means "use the default from netsim config.json".
	NetsimMode string `json:"netsimMode"`

	// NetsimMaxRangeM overrides max_range_m in netsim config when NetsimMode is "fixed_range".
	// Nil means "use the default from netsim config.json".
	NetsimMaxRangeM *float64 `json:"netsimMaxRangeM,omitempty"`
}

// SanitizeSimulationName ensures the simulation name is filesystem-friendly.
func (c *GeneralConfig) SanitizeSimulationName() {
	name := strings.TrimSpace(c.SimulationName)
	if name == "" {
		name = time.Now().Format("20060102_150405")
	} else {
		reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]+`)
		name = reg.ReplaceAllString(name, "_")
	}
	c.SimulationName = name
}
