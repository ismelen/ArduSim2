package domain

// Location3DUTM represents a 3D coordinate in UTM (Universal Transverse Mercator)
type Location3DUTM struct {
	X float64 // Easting
	Y float64 // Northing
	Z float64 // Altitude
}

func (l Location3DUTM) Distance(other Location3DUTM) float64 {
	// Not implementing the Math here, this is just a struct, but we could logic it
	// Clean Arch limits logic in domain usually, but basic math is fine.
	return 0 // Math implemented in usecase
}

// Location2DUTM represents 2D coordinates
type Location2DUTM struct {
	X float64
	Y float64
}

// MBCAPState represents the states of the FSM
type MBCAPState int

const (
	NORMAL MBCAPState = iota
	STAND_STILL
	MOVING_ASIDE
	GO_ON_PLEASE
	OVERTAKING
	EMERGENCY_LAND
)

// Beacon represents the telemetry packet exchanged by drones for MBCAP
type Beacon struct {
	UavID      int64
	State      MBCAPState
	IsLanding  bool
	Speed      float64
	Time       int64 // timestamp in nanoseconds
	Points     []Location3DUTM
	Event      int
	IdAvoiding int64
}

// Defaults
const ID_AVOIDING_DEFAULT int64 = -1

// AppConfig represents configuration variables from the file or environment
type AppConfig struct {
	UavID             int64  `json:"uav_id"`
	BrokerIP          string `json:"broker_ip"`
	BrokerPort        int    `json:"broker_port"`
	TelemetryTopic    string `json:"telemetry_topic"`
	ExternalTelemetry string `json:"external_telemetry"`
	CmdPublishTopic   string `json:"cmd_publish_topic"`
	LogsTopic         string `json:"logs_topic"`

	// Algorithmic parameters previously from mbcap.properties
	CollisionWarningDistance       float64 `json:"collisionWarningDistance"`
	CollisionWarningAltitudeOffset float64 `json:"collisionWarningAltitudeOffset"`
	CollisionWarningTimeOffset     float64 `json:"collisionWarningTimeOffset"`
	RiskCheckPeriod                float64 `json:"riskCheckPeriod"`
	BeaconExpirationTime           float64 `json:"beaconExpirationTime"`
	HoveringTimeout                float64 `json:"hoveringTimeout"`
	DefaultFlightModeResumeDelay   float64 `json:"defaultFlightModeResumeDelay"`
	CheckRiskSameUAVDelay          float64 `json:"checkRiskSameUAVDelay"`
	OvertakeDelayTimeout           float64 `json:"overtakeDelayTimeout"`
	DeadlockBaseTimeout            int64   `json:"deadlockBaseTimeout"`
	SafePlaceDistance              float64 `json:"safePlaceDistance"`
	SafetyDistanceRange            float64 `json:"safetyDistanceRange"`
	HopTimeNS                      int64   `json:"hopTimeNS"`
	MinAdvertismentSpeed           float64 `json:"minAdvertismentSpeed"`
}

// MBCAPParam stores constants for MBCAP algorithm
type MBCAPParam struct {
	CollisionWarningDistance       float64
	CollisionWarningAltitudeOffset float64
	CollisionWarningTimeOffset     int64
	RiskCheckPeriod                int64
	BeaconExpirationTime           int64
	HoveringTimeout                int64
	DefaultFlightModeResumeDelay   int64
	CheckRiskSameUAVDelay          int64
	OvertakeDelayTimeout           int64
	DeadlockBaseTimeout            int64
	SafePlaceDistance              float64
	SafetyDistanceRange            float64
	HopTimeNS                      int64
	MinAdvertismentSpeed           float64
}
