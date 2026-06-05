package domain

// UAV groups a UAV identifier with its set of deployed services.
type UAV struct {
	ID           string            `json:"id"`
	Services     []DeployedService `json:"services"`
	Speed        *float64          `json:"speed,omitempty"`
	HomeOverride *Coordinate       `json:"homeOverride,omitempty"`
}
