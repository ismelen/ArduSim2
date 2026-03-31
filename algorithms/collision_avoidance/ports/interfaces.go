package ports

import "collision_avoidance/domain"

// Broker is an interface for pub/sub operations.
type Broker interface {
	Connect(ip string, port int, subTopics []string) error
	Publish(topic string, payload map[string]interface{}) error
	Listen() (<-chan BrokerMessage, error)
	Close() error
}

type BrokerMessage struct {
	Topic   string
	Payload map[string]interface{}
}

// TelemetryStore maintains the latest telemetry status of drones
type TelemetryStore interface {
	UpdateExternalBeacon(beacon domain.Beacon)
	GetExternalBeacons() map[int64]domain.Beacon
	UpdateOwnBeacon(beacon domain.Beacon)
	GetOwnBeacon() *domain.Beacon
}

// ConfigLoader is an interface for reading the application configuration.
type ConfigLoader interface {
	LoadAppConfig(filePath string) (*domain.AppConfig, error)
	LoadMBCAPParams(filePath string) (*domain.MBCAPParam, error)
}
