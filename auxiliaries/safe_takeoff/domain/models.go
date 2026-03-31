package domain

import (
	"time"
)

type ActionType string

const (
	ActionTakeoff        ActionType = "Takeoff"
	ActionMoveToPosition ActionType = "MoveToPosition"
)

// AppConfig matches config.json
type AppConfig struct {
	SelfUAVId             int     `json:"self_uav_id"`
	MasterID              int     `json:"master_id"`
	NumUAVs               int     `json:"num_uavs"`
	BrokerIP              string  `json:"broker_ip"`
	BrokerPort            int     `json:"broker_port"`
	TelemetryTopic        string  `json:"telemetry_topic"`
	ExternalMessagesTopic string  `json:"external_messages_topic"`
	SuggestionsTopic      string  `json:"suggestions_topic"`
	GroundFormation       string  `json:"ground_formation"`
	FlyingFormation       string  `json:"flying_formation"`
	GroundMinDistance     float64 `json:"ground_min_distance"`
	FlyingMinDistance     float64 `json:"flying_min_distance"`
	TakeOffStrategy       string  `json:"takeoff_strategy"` // "Hungarian", "Simplified"
	TakeOffIsSequential   bool    `json:"takeoff_is_sequential"`
	Altitude              float64 `json:"altitude"`
}

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Alt float64 `json:"alt"`
}

// ProtocolMsg matches the payload for external/messages
type ProtocolMsg struct {
	Type     string      `json:"type"` // "HELLO", "TAKEOFF_DATA", "ACK", "GO"
	SenderID int         `json:"sender_id"`
	Payload  interface{} `json:"payload,omitempty"`
}

// HelloPayload is sent by slaves during discovery
type HelloPayload struct {
	Location Location `json:"location"`
}

// TakeOffDataPayload is sent by the master to assign targets
type TakeOffDataPayload struct {
	Assignments map[int]Location `json:"assignments"` // UAVId -> TargetLocation
}

// GoPayload triggers the takeoff for a specific UAV
type GoPayload struct {
	TargetUAV int `json:"target_uav"`
}

// TelemetryMessage from internal broker
type TelemetryMessage struct {
	UAVId     int       `json:"uav_id"`
	Location  Location  `json:"location"`
	Timestamp time.Time `json:"timestamp"`
}

// Suggestion for the application module
type Suggestion struct {
	Endpoint  ActionType `json:"endpoint"`
	Latitude  float64    `json:"latitude,omitempty"`
	Longitude float64    `json:"longitude,omitempty"`
	Altitude  float64    `json:"altitude,omitempty"`
}
