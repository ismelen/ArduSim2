package ports

import "safe_takeoff/domain"

type ConfigLoader interface {
	LoadAppConfig(path string) (*domain.AppConfig, error)
}

type Broker interface {
	Connect(address string) error
	Publish(topic string, payload []byte) error
	Subscribe(topic string, handler func([]byte)) error
	Close() error
}
