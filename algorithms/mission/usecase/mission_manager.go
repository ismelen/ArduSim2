package usecase

import (
	"log"
	"math"

	"mission/domain"
	"mission/ports"
)

type MissionManager struct {
	broker        ports.Broker
	configLoader  ports.ConfigLoader
	missionParser ports.MissionParser

	config    *domain.AppConfig
	waypoints []domain.Waypoint

	state          domain.State
	currentWpIndex int
}

func NewMissionManager(b ports.Broker, c ports.ConfigLoader, p ports.MissionParser) *MissionManager {
	return &MissionManager{
		broker:        b,
		configLoader:  c,
		missionParser: p,
		state:         domain.IDLE,
	}
}

func (m *MissionManager) Initialize(configFile, kmlFile string) error {
	var err error
	m.config, err = m.configLoader.LoadAppConfig(configFile)
	if err != nil {
		return err
	}

	m.waypoints, err = m.missionParser.ParseMission(kmlFile)
	if err != nil {
		return err
	}

	if len(m.waypoints) > 1 {
		m.currentWpIndex = 1
	}

	return m.broker.Connect(m.config.BrokerIP, m.config.BrokerPort, m.config.SubscriptionTopic, m.config.TelemetryTopic)
}

func (m *MissionManager) Run() {
	msgChan, err := m.broker.Listen()
	if err != nil {
		log.Fatalf("Failed to listen to broker: %v", err)
	}

	log.Println("Mission Manager initialized. Waiting for commands...")

	for msg := range msgChan {
		m.handleMessage(msg)
	}
}

func (m *MissionManager) handleMessage(msg domain.BrokerMessage) {
	if msg.Topic == m.config.SubscriptionTopic {
		m.handleCommand(msg.Payload)
	} else if msg.Topic == m.config.TelemetryTopic {
		m.handleTelemetry(msg.Payload)
	}
}

const (
	// Commands from ArduSim PC Companion
	CommandChangeState = 1
	CommandEmergency   = 2

	// Emergency Actions
	ActionNone           = 1
	ActionRecoverControl = 2
	ActionRTL            = 3
	ActionLand           = 4
)

func (m *MissionManager) handleCommand(payload map[string]interface{}) {
	cmdFloat, ok := payload["command"].(float64)
	if !ok {
		return
	}
	cmd := int(cmdFloat)

	switch cmd {
	case CommandChangeState: // Equivalent to "start"
		if m.state == domain.IDLE || m.state == domain.PAUSED {
			if m.state == domain.IDLE {
				m.broker.Publish(m.config.PublishTopic, map[string]interface{}{"endpoint": "Arm"})
				m.broker.Publish(m.config.PublishTopic, map[string]interface{}{"endpoint": "SetFlightmode", "flightmode": "GUIDED"})

				takeoffAlt := m.waypoints[1].Altitude
				if takeoffAlt == 0 {
					takeoffAlt = m.config.MinimumWaypointRelativeAltitude
				}

				m.broker.Publish(m.config.PublishTopic, map[string]interface{}{
					"endpoint": "Takeoff",
					"altitude": takeoffAlt,
				})
				m.state = domain.TAKEOFF
				log.Printf("Starting mission: taking off to %.2f\n", takeoffAlt)
			} else {
				m.state = domain.FLYING
				log.Println("Resuming mission...")
				m.broker.Publish(m.config.PublishTopic, map[string]interface{}{"endpoint": "SetFlightmode", "flightmode": "GUIDED"})
			}
		}
	case CommandEmergency:
		actionFloat, ok := payload["action"].(float64)
		if !ok {
			return
		}
		action := int(actionFloat)

		switch action {
		case ActionRecoverControl: // Equivalent to "pause"
			if m.state == domain.FLYING {
				m.state = domain.PAUSED
				m.broker.Publish(m.config.PublishTopic, map[string]interface{}{"endpoint": "SetFlightmode", "flightmode": "BRAKE"})
				log.Println("Mission paused (Recover Control).")
			}
		case ActionRTL: // Equivalent to "returnToHome"
			m.state = domain.LANDING
			m.broker.Publish(m.config.PublishTopic, map[string]interface{}{"endpoint": "SetFlightmode", "flightmode": "RTL"})
			log.Println("Returning to Home (RTL)...")
		case ActionLand: // Equivalent to "stop"
			m.state = domain.LANDING
			m.broker.Publish(m.config.PublishTopic, map[string]interface{}{"endpoint": "Land"})
			log.Println("Emergency Land...")
		}
	}
}

func (m *MissionManager) handleTelemetry(payload map[string]interface{}) {
	switch m.state {
	case domain.TAKEOFF:
		alt, _ := payload["relative_alt"].(float64)
		takeoffAlt := m.waypoints[1].Altitude
		if takeoffAlt == 0 {
			takeoffAlt = m.config.MinimumWaypointRelativeAltitude
		}
		if math.Abs(alt-takeoffAlt) < 1.0 {
			m.state = domain.FLYING
			m.currentWpIndex = 2
			m.sendNextWaypoint()
		}
	case domain.FLYING:
		lat, ok1 := payload["lat"].(float64)
		lon, ok2 := payload["lon"].(float64)

		if ok1 && ok2 && m.currentWpIndex < len(m.waypoints) {
			target := m.waypoints[m.currentWpIndex]
			dist := haversine(lat, lon, target.Latitude, target.Longitude)

			if dist < m.config.DistanceToWaypointReached {
				log.Printf("Reached Waypoint %d\n", m.currentWpIndex)
				m.currentWpIndex++
				if m.currentWpIndex < len(m.waypoints) {
					m.sendNextWaypoint()
				} else {
					m.state = domain.LANDING
					m.broker.Publish(m.config.PublishTopic, map[string]interface{}{"endpoint": "Land"})
					log.Println("Mission complete. Landing...")
				}
			}
		}
	}
}

func (m *MissionManager) sendNextWaypoint() {
	if m.currentWpIndex < len(m.waypoints) {
		wp := m.waypoints[m.currentWpIndex]
		m.broker.Publish(m.config.PublishTopic, map[string]interface{}{
			"endpoint":  "MoveToPosition",
			"latitude":  wp.Latitude,
			"longitude": wp.Longitude,
			"altitude":  m.config.WaypointsRelativeAltitude,
		})
		log.Printf("Moving to Waypoint %d\n", m.currentWpIndex)
	}
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
