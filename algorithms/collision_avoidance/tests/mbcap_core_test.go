package tests

import (
	"testing"
	"time"

	"collision_avoidance/domain"
	"collision_avoidance/infrastructure"
	"collision_avoidance/ports"
	"collision_avoidance/usecase"
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

func (m *MockBroker) Connect(ip string, port int, subTopics []string) error {
	return nil
}

func (m *MockBroker) Publish(topic string, payload map[string]interface{}) error {
	m.PublishedMessages = append(m.PublishedMessages, ports.BrokerMessage{
		Topic:   topic,
		Payload: payload,
	})
	return nil
}

func (m *MockBroker) Listen() (<-chan ports.BrokerMessage, error) {
	return m.msgChan, nil
}

func (m *MockBroker) Close() error {
	close(m.msgChan)
	return nil
}

func TestCollisionRiskDetection(t *testing.T) {
	// Setup dependencies
	mockBroker := NewMockBroker()

	configLoader := infrastructure.NewFileLoader()
	config, params, err := configLoader.LoadAppConfig("data/test_config.json")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	store := infrastructure.NewMemoryTelemetryStore(5000000000)

	core := usecase.NewMBCAPCore(mockBroker, store, config, params)

	// Self Beacon (Standing around origin)
	selfBeacon := domain.Beacon{
		UavID:  1,
		State:  domain.NORMAL,
		Speed:  1.0,
		Time:   time.Now().UnixNano(),
		Points: []domain.Location3DUTM{{X: 0, Y: 0, Z: 10}, {X: 1, Y: 1, Z: 10}}, // Moving very little
	}
	store.UpdateOwnBeacon(selfBeacon)

	// External Beacon (Also normal, approaching origin)
	extBeacon := domain.Beacon{
		UavID:  2,
		State:  domain.NORMAL,
		Speed:  5.0,
		Time:   time.Now().UnixNano(),
		Points: []domain.Location3DUTM{{X: 5, Y: 5, Z: 10}, {X: 0, Y: 0, Z: 10}}, // Moves right into the origin in 1 second
	}
	store.UpdateExternalBeacon(extBeacon)

	// 1st Evaluation -> Detect risk -> Transition to STAND_STILL
	core.EvaluateCollisionRisk()

	// Fetch updated own beacon
	updatedSelf := store.GetOwnBeacon()
	if updatedSelf.State != domain.STAND_STILL {
		t.Fatalf("Expected state to transition from NORMAL to STAND_STILL when risk detected, got: %v", updatedSelf.State)
	}

	if updatedSelf.IdAvoiding != 2 {
		t.Errorf("Expected to be avoiding UAV 2, got %v", updatedSelf.IdAvoiding)
	}
}
