package ports

import "external_comms/domain"

// Broker handles pub/sub events locally (Docker network)
type Broker interface {
	Connect(ip string, port int, createTopics []string) error
	Publish(topic string, payload map[string]interface{}) error
	Listen() (<-chan BrokerMessage, error)
	Close() error
}

type BrokerMessage struct {
	Topic   string
	Payload map[string]interface{}
}

// NetSimLink handles raw UDP communication with the external Network Simulator
type NetSimLink interface {
	Send(msg domain.NetSimMessage) error
	Listen() (<-chan domain.NetSimMessage, error)
	Close() error
}
