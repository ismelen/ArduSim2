package tests

import (
	"encoding/json"
	"testing"
	"time"

	"application/domain"
	"application/infrastructure"
	"application/ports"
	"application/usecase"
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
func (m *MockBroker) SendToChannel(s domain.Suggestion) {
	data, _ := json.Marshal(s)
	var payload map[string]interface{}
	json.Unmarshal(data, &payload)
	m.msgChan <- ports.BrokerMessage{Topic: "uav/suggestions", Payload: payload}
}

type MockUAVLink struct {
	SentSuggestions []domain.Suggestion
	telChan         chan map[string]interface{}
}

func NewMockUAVLink() *MockUAVLink {
	return &MockUAVLink{
		telChan: make(chan map[string]interface{}, 10),
	}
}

func (m *MockUAVLink) SendSuggestion(s domain.Suggestion) error {
	m.SentSuggestions = append(m.SentSuggestions, s)
	return nil
}
func (m *MockUAVLink) ListenTelemetry() (<-chan map[string]interface{}, error) {
	return m.telChan, nil
}

func (m *MockUAVLink) Close() error { return nil }

func TestMovementMixerPriorityOverride(t *testing.T) {
	broker := NewMockBroker()
	uav := NewMockUAVLink()

	configLoader := infrastructure.NewFileLoader()
	config, err := configLoader.LoadAppConfig("data/test_config.json")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	mixer := usecase.NewMovementMixer(broker, uav, config)

	// Create suggestions
	sugVector := domain.Suggestion{Endpoint: domain.ActionMoveToPosition, Latitude: 10.0}
	sugStructural := domain.Suggestion{Endpoint: domain.ActionSetFlightmode, Flightmode: "RTL"}

	go mixer.Run()

	// Feed them in the window
	broker.SendToChannel(sugVector)
	broker.SendToChannel(sugStructural)

	// Wait for window
	time.Sleep(200 * time.Millisecond)

	if len(uav.SentSuggestions) != 1 {
		t.Fatalf("Expected 1 suggestion sent to UAV, got %d", len(uav.SentSuggestions))
	}

	if uav.SentSuggestions[0].Endpoint != domain.ActionSetFlightmode {
		t.Errorf("Expected Structural priority to override, but got %v", uav.SentSuggestions[0].Endpoint)
	}
}

func TestMovementMixerVectorMix(t *testing.T) {
	broker := NewMockBroker()
	uav := NewMockUAVLink()

	configLoader := infrastructure.NewFileLoader()
	config, err := configLoader.LoadAppConfig("data/test_config.json")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	mixer := usecase.NewMovementMixer(broker, uav, config)

	// Vector A
	sugVectorA := domain.Suggestion{Endpoint: domain.ActionMoveToPosition, Latitude: 10.0, Longitude: 0.0}
	// Vector B
	sugVectorB := domain.Suggestion{Endpoint: domain.ActionMoveToPosition, Latitude: 0.0, Longitude: 20.0}

	go mixer.Run()

	// Feed them
	broker.SendToChannel(sugVectorA)
	broker.SendToChannel(sugVectorB)

	// Wait for window
	time.Sleep(200 * time.Millisecond)

	if len(uav.SentSuggestions) != 1 {
		t.Fatalf("Expected 1 combined suggestion sent to UAV, got %d", len(uav.SentSuggestions))
	}

	result := uav.SentSuggestions[0]
	if result.Endpoint != domain.ActionMoveToPosition {
		t.Errorf("Expected move to position, but got %v", result.Endpoint)
	}
	if result.Latitude != 5.0 || result.Longitude != 10.0 { // Averaged
		t.Errorf("Expected Averaged vector (5.0, 10.0), got (%f, %f)", result.Latitude, result.Longitude)
	}
}
