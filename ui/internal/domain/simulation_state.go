package domain

// SimulationState captures the full UI state at the time a simulation starts.
// This is used to "Load" a previous simulation exactly as it was configured.
type SimulationState struct {
	UAVs          []UAV         `json:"uavs"`
	GeneralConfig GeneralConfig `json:"generalConfig"`
	ActiveMode    string        `json:"activeMode"`
}
