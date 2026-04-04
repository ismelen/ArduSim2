package domain

// Waypoint represents a geographic location parsed from a mission file.
type Waypoint struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
}

// State represents the current state of the mission algorithm.
type State int

const (
	IDLE State = iota
	TAKEOFF
	FLYING
	PAUSED
	LANDING
)

// AppConfig represents the JSON execution configuration.
type AppConfig struct {
	MissionFile                     string  `json:"mission_file"`
	BrokerIP                        string  `json:"broker_ip"`
	BrokerPort                      int     `json:"broker_port"`
	SubscriptionTopic               string  `json:"subscription_topic"`
	PublishTopic                    string  `json:"publish_topic"`
	TelemetryTopic                  string  `json:"telemetry_topic"`
	DistanceToWaypointReached       float64 `json:"distance_to_waypoint_reached"`
	MinimumWaypointRelativeAltitude float64 `json:"minimum_waypoint_relative_altitude"`
	WaypointsRelativeAltitude       float64 `json:"waypoints_relative_altitude"`
}

// BrokerMessage represents a message received from or sent to the broker.
type BrokerMessage struct {
	Topic   string
	Payload map[string]interface{}
}