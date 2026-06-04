package domain

type SimulationState struct {
	UAVs          []UAV         `json:"uavs"`
	GeneralConfig GeneralConfig `json:"generalConfig"`
	ActiveMode    string        `json:"activeMode"`
}
