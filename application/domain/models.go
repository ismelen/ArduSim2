package domain

type ActionType string

const (
	ActionMoveToPosition ActionType = "MoveToPosition"
	ActionMoveByVector   ActionType = "MoveByVector"
	ActionSetFlightmode  ActionType = "SetFlightmode"
	ActionLand           ActionType = "Land"
	ActionTakeoff        ActionType = "Takeoff"
	ActionRotate         ActionType = "Rotate"
)

// Suggestion represents a JSON command payload arriving from an algorithm
type Suggestion struct {
	Endpoint   ActionType `json:"endpoint"`
	Latitude   float64    `json:"latitude,omitempty"`
	Longitude  float64    `json:"longitude,omitempty"`
	Altitude   float64    `json:"altitude,omitempty"`
	Flightmode string     `json:"flightmode,omitempty"`
}

// AppConfig is the initialization struct pointing to the controller and the broker
type AppConfig struct {
	BrokerIP          string `json:"broker_ip"`
	BrokerPort        int    `json:"broker_port"`
	UAVControllerIP   string `json:"uav_controller_ip"`
	UAVControllerPort int    `json:"uav_controller_port"`
	UAVTelemetryPort  int    `json:"uav_telemetry_port"`
	
	TelemetryTopic    string `json:"telemetry_topic"`
	SuggestionsTopic  string `json:"suggestions_topic"`
	SwarmTelemetry    string `json:"external_telemetry"`
	SwarmMessages     string `json:"external_messages"`
	GlobalCommands    string `json:"global_commands"`
	
	MixWindowMs       int    `json:"mix_window_ms"` // The time window (ms) to collect and merge suggestions 
}

// IsStructural returns true if the suggestion implies halting standard vector merging
func (s Suggestion) IsStructural() bool {
	return s.Endpoint == ActionSetFlightmode || s.Endpoint == ActionLand || s.Endpoint == ActionTakeoff
}
