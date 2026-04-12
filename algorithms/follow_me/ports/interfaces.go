package ports

type BrokerMessage struct {
	Topic   string      `json:"topic"`
	Payload interface{} `json:"payload"`
}

type CommunicationProvider interface {
	Connect(ip string, port int, topics ...string) error
	Publish(topic string, payload interface{}) error
	Listen() (<-chan BrokerMessage, error)
	Close() error
}

type FollowMeManager interface {
	Run() error
}
