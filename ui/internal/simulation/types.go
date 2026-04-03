package simulation

// ServiceType describes an available algorithm/service discovered on disk.
type ServiceType struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SchemaRaw string `json:"schemaRaw"`
}

// DeployedService represents a user-selected service assigned to a UAV,
// including its resolved runtime configuration.
type DeployedService struct {
	InstanceId   string                 `json:"instanceId"`
	ServiceId    string                 `json:"serviceId"`
	ServiceTitle string                 `json:"serviceTitle"`
	Config       map[string]interface{} `json:"config"`
}

// GeneralConfig contains simulation-wide parameters like wind and battery.
type GeneralConfig struct {
	SpeedProfilePath  string  `json:"speedProfilePath"`
	LoggingEnabled    bool    `json:"loggingEnabled"`
	BatteryRestricted bool    `json:"batteryRestricted"`
	BatteryCapacity   int     `json:"batteryCapacity"`
	VerboseLogging    bool    `json:"verboseLogging"`
	StoreLocalData    bool    `json:"storeLocalData"`
	WindEnabled       bool    `json:"windEnabled"`
	WindDirection     float64 `json:"windDirection"`
	WindSpeed         float64 `json:"windSpeed"`
}

// UAV groups a UAV identifier with its set of deployed services.
type UAV struct {
	ID       string            `json:"id"`
	Services []DeployedService `json:"services"`
}

// SimulationState captures the full UI state at the time a simulation starts.
// This is used to "Load" a previous simulation exactly as it was configured.
type SimulationState struct {
	UAVs          []UAV         `json:"uavs"`
	GeneralConfig GeneralConfig `json:"generalConfig"`
	ActiveMode    string        `json:"activeMode"`
}
