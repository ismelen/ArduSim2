package tests

import (
	"testing"
	"time"

	"external_comms/domain"
	"external_comms/infrastructure"
	"external_comms/ports"
	"external_comms/usecase"
)

type MockBroker struct {
	PublishedMessages []ports.BrokerMessage
	msgChan           chan ports.BrokerMessage
}

func NewMockBroker() *MockBroker {
	return &MockBroker{
		msgChan: make(chan ports.BrokerMessage, 10),
	}
}

func (m *MockBroker) Connect(ip string, port int, subTopics []string) error { return nil }
func (m *MockBroker) Publish(topic string, payload map[string]interface{}) error {
	m.PublishedMessages = append(m.PublishedMessages, ports.BrokerMessage{Topic: topic, Payload: payload})
	return nil
}
func (m *MockBroker) Listen() (<-chan ports.BrokerMessage, error) { return m.msgChan, nil }
func (m *MockBroker) Close() error                                { close(m.msgChan); return nil }
func (m *MockBroker) SendToChannel(topic string, payload map[string]interface{}) {
	m.msgChan <- ports.BrokerMessage{Topic: topic, Payload: payload}
}

type MockNetSimLink struct {
	SentMessages []domain.NetSimMessage
	msgChan      chan domain.NetSimMessage
}

func NewMockNetSimLink() *MockNetSimLink {
	return &MockNetSimLink{
		msgChan: make(chan domain.NetSimMessage, 10),
	}
}

func (m *MockNetSimLink) Send(msg domain.NetSimMessage) error {
	m.SentMessages = append(m.SentMessages, msg)
	return nil
}
func (m *MockNetSimLink) Listen() (<-chan domain.NetSimMessage, error) { return m.msgChan, nil }
func (m *MockNetSimLink) Close() error                                 { close(m.msgChan); return nil }
func (m *MockNetSimLink) SendToChannel(msg domain.NetSimMessage) {
	m.msgChan <- msg
}

func TestGatewayBridge_InternalToExternal(t *testing.T) {
	broker := NewMockBroker()
	netSim := NewMockNetSimLink()

	configLoader := infrastructure.NewFileLoader()
	config, _ := configLoader.LoadAppConfig("data/test_config.json")

	bridge := usecase.NewGatewayBridge(config, broker, netSim)

	go bridge.Run()

	// Simulate receiving telemetry from the local broker
	payload := map[string]interface{}{"lat": 10.0}
	broker.SendToChannel("telemetry", payload)

	time.Sleep(100 * time.Millisecond)

	if len(netSim.SentMessages) != 1 {
		t.Fatalf("Expected 1 message forwarded to netSim, got %d", len(netSim.SentMessages))
	}

	res := netSim.SentMessages[0]
	if res.Topic != "telemetry" || res.Source != "1" {
		t.Errorf("Expected external telemetry msg from source 1, got type=%s source=%s", res.Topic, res.Source)
	}
	if res.Payload["lat"] != 10.0 {
		t.Errorf("Payload mismatch")
	}
}

func TestGatewayBridge_ExternalToInternal(t *testing.T) {
	broker := NewMockBroker()
	netSim := NewMockNetSimLink()

	configLoader := infrastructure.NewFileLoader()
	config, _ := configLoader.LoadAppConfig("data/test_config.json")

	bridge := usecase.NewGatewayBridge(config, broker, netSim)

	go bridge.Run()

	// Simulate receiving external message from drone 2
	netMsg := domain.NetSimMessage{
		Topic:    "message",
		Source:  "2",
		Payload: map[string]interface{}{"text": "hello"},
	}
	netSim.SendToChannel(netMsg)

	time.Sleep(100 * time.Millisecond)

	if len(broker.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 message published to internal broker, got %d", len(broker.PublishedMessages))
	}

	res := broker.PublishedMessages[0]
	if res.Topic != "external/messages" {
		t.Errorf("Expected external/messages topic, got %s", res.Topic)
	}
	if res.Payload["text"] != "hello" {
		t.Errorf("Payload mismatch")
	}
}
