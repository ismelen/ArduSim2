package usecase

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"application/domain"
	"application/ports"
)

type MovementMixer struct {
	broker ports.Broker
	uav    ports.UAVLink
	config *domain.AppConfig

	mu           sync.Mutex
	suggestions  []domain.Suggestion
}

func NewMovementMixer(b ports.Broker, u ports.UAVLink, c *domain.AppConfig) *MovementMixer {
	return &MovementMixer{
		broker: b,
		uav:    u,
		config: c,
	}
}

func (m *MovementMixer) Run() {
	msgChan, err := m.broker.Listen()
	if err != nil {
		log.Fatalf("Failed to listen on broker: %v", err)
	}

	ticker := time.NewTicker(time.Duration(m.config.MixWindowMs) * time.Millisecond)
	defer ticker.Stop()

	log.Println("Movement Mixer Started...")

	// Telemetry Bridge
	telemetryChan, err := m.uav.ListenTelemetry()
	if err != nil {
		log.Printf("Failed to listen for telemetry: %v", err)
	}

	for {
		select {
		case tel := <-telemetryChan:
			m.broker.Publish(m.config.TelemetryTopic, tel)
		case msg := <-msgChan:
			if msg.Topic == m.config.SuggestionsTopic {
				m.handleSuggestionArrival(msg.Payload)
			}
		case <-ticker.C:
			m.evaluateAndMixWindow()
		}
	}
}

func (m *MovementMixer) handleSuggestionArrival(payload map[string]interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	var sug domain.Suggestion
	if err := json.Unmarshal(data, &sug); err == nil {
		m.mu.Lock()
		m.suggestions = append(m.suggestions, sug)
		m.mu.Unlock()
	}
}

func (m *MovementMixer) evaluateAndMixWindow() {
	m.mu.Lock()
	currentBatch := m.suggestions
	m.suggestions = nil // reset window
	m.mu.Unlock()

	if len(currentBatch) == 0 {
		return
	}

	// 1. Check for Absolute Priority (Structural overrules all)
	for _, req := range currentBatch {
		if req.IsStructural() {
			log.Printf("Executing Structural Priority Override: %v", req.Endpoint)
			m.uav.SendSuggestion(req)
			
			// If it's a stopping event, broadcast to others
			if req.Endpoint == domain.ActionLand || req.Flightmode == "RTL" || req.Flightmode == "BRAKE" {
				m.broker.Publish(m.config.GlobalCommands, map[string]interface{}{
					"command": "stop",
				})
			}
			return // Ignore any other vector
		}
	}

	// 2. Vectorial Mix (If no structural priority occurred)
	// For simplicity, we split into Point-based and Vector-based. 
	// If the batch has conflicts (some want Points, some want Vectors), we prioritize the most frequent.
	// For this naive mix, we will just average Lat/Lon/Alt for MoveToPosition.
	
	var latSum, lonSum, altSum float64
	var count float64
	var lastType domain.ActionType

	for _, req := range currentBatch {
		if req.Endpoint == domain.ActionMoveToPosition || req.Endpoint == domain.ActionMoveByVector {
			latSum += req.Latitude
			lonSum += req.Longitude
			altSum += req.Altitude
			count++
			lastType = req.Endpoint
		}
	}

	if count > 0 {
		mixed := domain.Suggestion{
			Endpoint:  lastType,
			Latitude:  latSum / count,
			Longitude: lonSum / count,
			Altitude:  altSum / count,
		}
		
		log.Printf("Executing Mixed Vector (%d suggestions): Type=%v Lat=%f Lon=%f", int(count), mixed.Endpoint, mixed.Latitude, mixed.Longitude)
		m.uav.SendSuggestion(mixed)
	}
}
