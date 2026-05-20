package infrastructure

import (
	"os"
	"strings"
	"time"

	"mission/ports"
)

type BrokerLogWriter struct {
	broker ports.Broker
	topic  string
}

func NewBrokerLogWriter(broker ports.Broker, topic string) *BrokerLogWriter {
	return &BrokerLogWriter{
		broker: broker,
		topic:  topic,
	}
}

func (w *BrokerLogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	payload := map[string]interface{}{
		"InstanceID": hostname,
		"ServiceID":  "mission",
		"Level":      "INFO",
		"Timestamp":  time.Now().Format(time.RFC3339),
		"Message":    msg,
	}

	// Fire and forget
	w.broker.Publish(w.topic, payload)

	return len(p), nil
}
