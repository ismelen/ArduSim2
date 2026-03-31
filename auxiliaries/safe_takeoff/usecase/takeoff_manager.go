package usecase

import (
	"encoding/json"
	"log"
	"safe_takeoff/domain"
	"safe_takeoff/ports"
	"sync"
	"time"
)

type TakeOffManager struct {
	config *domain.AppConfig
	broker ports.Broker

	isMaster bool

	// Master state
	peers     map[int]domain.Location
	peerAcks  map[int]bool
	peerMutex sync.RWMutex

	// Slave state
	targetPos  *domain.Location
	myPosition *domain.Location
	posMutex   sync.RWMutex
}

func NewTakeOffManager(broker ports.Broker, envConfig *domain.AppConfig) *TakeOffManager {
	return &TakeOffManager{
		config:   envConfig,
		broker:   broker,
		isMaster: envConfig.SelfUAVId == envConfig.MasterID,
		peers:    make(map[int]domain.Location),
		peerAcks: make(map[int]bool),
	}
}

func (m *TakeOffManager) Start() {
	// 1. Subscribe to own telemetry
	err := m.broker.Subscribe(m.config.TelemetryTopic, m.handleOwnTelemetry)
	if err != nil {
		log.Fatalf("Failed to subscribe to telemetry: %v", err)
	}

	// 2. Subscribe to external coordination messages
	err = m.broker.Subscribe(m.config.ExternalMessagesTopic, m.handleProtocolMessage)
	if err != nil {
		log.Fatalf("Failed to subscribe to external messages: %v", err)
	}

	log.Printf("Safe TakeOff [%d] started as %s", m.config.SelfUAVId, m.roleName())

	if !m.isMaster {
		// Slave: Periodic Hello until discovery
		go m.discoveryLoop()
	} else {
		// Master: Monitor peers and transition
		go m.masterOrchestrationLoop()
	}

	select {} // Keep running
}

func (m *TakeOffManager) roleName() string {
	if m.isMaster {
		return "MASTER"
	}
	return "SLAVE"
}

// handleOwnTelemetry updates the UAV's current ground position
func (m *TakeOffManager) handleOwnTelemetry(payload []byte) {
	var tMsg domain.TelemetryMessage
	if err := json.Unmarshal(payload, &tMsg); err == nil {
		m.posMutex.Lock()
		m.myPosition = &tMsg.Location
		m.posMutex.Unlock()

		// Also add self to peers if master
		if m.isMaster {
			m.peerMutex.Lock()
			m.peers[m.config.SelfUAVId] = tMsg.Location
			m.peerMutex.Unlock()
		}
	}
}

// handleProtocolMessage handles inter-UAV coordination
func (m *TakeOffManager) handleProtocolMessage(payload []byte) {
	var msg domain.ProtocolMsg
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	// Ignore self-messages
	if msg.SenderID == m.config.SelfUAVId {
		return
	}

	switch msg.Type {
	case "HELLO":
		if m.isMaster {
			var h domain.HelloPayload
			data, _ := json.Marshal(msg.Payload)
			if err := json.Unmarshal(data, &h); err == nil {
				m.peerMutex.Lock()
				m.peers[msg.SenderID] = h.Location
				m.peerMutex.Unlock()
			}
		}
	case "TAKEOFF_DATA":
		if !m.isMaster {
			var d domain.TakeOffDataPayload
			data, _ := json.Marshal(msg.Payload)
			if err := json.Unmarshal(data, &d); err == nil {
				if target, ok := d.Assignments[m.config.SelfUAVId]; ok {
					m.posMutex.Lock()
					m.targetPos = &target
					m.posMutex.Unlock()
					log.Printf("Slave [%d] assigned to target location: %+v", m.config.SelfUAVId, target)
					m.sendAck()
				}
			}
		}
	case "ACK":
		if m.isMaster {
			m.peerMutex.Lock()
			m.peerAcks[msg.SenderID] = true
			m.peerMutex.Unlock()
		}
	case "GO":
		var g domain.GoPayload
		data, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(data, &g); err == nil {
			if g.TargetUAV == m.config.SelfUAVId {
				go m.executeTakeOffSequence()
			}
		}
	}
}

func (m *TakeOffManager) discoveryLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.posMutex.RLock()
		pos := m.myPosition
		m.posMutex.RUnlock()

		if pos != nil {
			m.broadcast(domain.ProtocolMsg{
				Type:     "HELLO",
				SenderID: m.config.SelfUAVId,
				Payload:  domain.HelloPayload{Location: *pos},
			})
		}
	}
}

func (m *TakeOffManager) masterOrchestrationLoop() {
	// 1. Wait for Discovery
	for {
		m.peerMutex.RLock()
		count := len(m.peers)
		m.peerMutex.RUnlock()

		if count >= m.config.NumUAVs {
			log.Printf("Master discovered all %d UAVs. Starting matching...", count)
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 2. Matching
	m.peerMutex.RLock()
	peersCopy := make(map[int]domain.Location)
	for id, loc := range m.peers {
		peersCopy[id] = loc
	}
	m.peerMutex.RUnlock()

	// Generate formation relative to master's position
	m.posMutex.RLock()
	masterPos := m.myPosition
	m.posMutex.RUnlock()

	if masterPos == nil {
		log.Println("Error: Master position unknown")
		return
	}

	var targets map[int]domain.Location
	switch m.config.FlyingFormation {
	case "Circle":
		targets = GenerateCircleFormation(m.config.NumUAVs, m.config.FlyingMinDistance)
	case "Matrix":
		targets = GenerateMatrixFormation(m.config.NumUAVs, m.config.FlyingMinDistance)
	case "Random":
		targets = GenerateRandomFormation(m.config.NumUAVs, m.config.FlyingMinDistance)
	default:
		targets = GenerateLinearFormation(m.config.NumUAVs, m.config.FlyingMinDistance)
	}

	// Adjust targets to master position (Lat/Lon)
	geoTargets := make(map[int]domain.Location)
	for i, t := range targets {
		geoTargets[i] = domain.Location{
			Lat: masterPos.Lat + (t.Lat / 111120.0),
			Lon: masterPos.Lon + (t.Lon / 111120.0),
			Alt: m.config.Altitude,
		}
	}

	assignments := MatchUAVsToFormation(peersCopy, geoTargets, m.config.TakeOffStrategy)

	// 3. Distribution
	log.Println("Master distributing TAKEOFF_DATA...")
	m.broadcast(domain.ProtocolMsg{
		Type:     "TAKEOFF_DATA",
		SenderID: m.config.SelfUAVId,
		Payload:  domain.TakeOffDataPayload{Assignments: assignments},
	})

	// 4. Wait for ACKs
	for {
		m.peerMutex.RLock()
		ackCount := len(m.peerAcks)
		m.peerMutex.RUnlock()

		// In distributed mode, we wait for all slaves (N-1)
		if ackCount >= m.config.NumUAVs-1 {
			log.Println("All slaves acknowledged. Starting execution...")
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 5. Execution (Sequential GO)
	for id := range assignments {
		log.Printf("Master signaling GO to UAV [%d]", id)
		m.broadcast(domain.ProtocolMsg{
			Type:     "GO",
			SenderID: m.config.SelfUAVId,
			Payload:  domain.GoPayload{TargetUAV: id},
		})

		if m.config.TakeOffIsSequential {
			time.Sleep(3 * time.Second) // Delay between drones
		}
	}
}

func (m *TakeOffManager) sendAck() {
	m.broadcast(domain.ProtocolMsg{
		Type:     "ACK",
		SenderID: m.config.SelfUAVId,
	})
}

func (m *TakeOffManager) executeTakeOffSequence() {
	m.posMutex.RLock()
	target := m.targetPos
	m.posMutex.RUnlock()

	if target == nil {
		return
	}

	log.Printf("Executing TakeOff sequence for UAV [%d]", m.config.SelfUAVId)

	// Phase 1: Takeoff
	m.suggest(domain.Suggestion{
		Endpoint: domain.ActionTakeoff,
		Altitude: m.config.Altitude,
	})

	time.Sleep(5 * time.Second) // Wait for altitude

	// Phase 2: Move to Target
	m.suggest(domain.Suggestion{
		Endpoint:  domain.ActionMoveToPosition,
		Latitude:  target.Lat,
		Longitude: target.Lon,
		Altitude:  target.Alt,
	})
}

func (m *TakeOffManager) broadcast(msg domain.ProtocolMsg) {
	payload, _ := json.Marshal(msg)
	m.broker.Publish(m.config.ExternalMessagesTopic, payload)
}

func (m *TakeOffManager) suggest(sug domain.Suggestion) {
	payload, _ := json.Marshal(sug)
	m.broker.Publish(m.config.SuggestionsTopic, payload)
}
