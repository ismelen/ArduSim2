package usecase

import (
	"log"
	"time"

	"collision_avoidance/domain"
	"collision_avoidance/ports"
)

type MBCAPCore struct {
	broker         ports.Broker
	store          ports.TelemetryStore
	config         *domain.AppConfig
	params         *domain.MBCAPParam
	currentState   domain.MBCAPState
	stateTime      int64
	idAvoiding     int64
	avoidingBeacon *domain.Beacon
}

func NewMBCAPCore(b ports.Broker, s ports.TelemetryStore, c *domain.AppConfig, p *domain.MBCAPParam) *MBCAPCore {
	return &MBCAPCore{
		broker:       b,
		store:        s,
		config:       c,
		params:       p,
		currentState: domain.NORMAL,
		stateTime:    time.Now().UnixNano(),
		idAvoiding:   domain.ID_AVOIDING_DEFAULT,
	}
}

func (m *MBCAPCore) Run() {
	ticker := time.NewTicker(time.Duration(m.params.RiskCheckPeriod))
	defer ticker.Stop()

	log.Println("MBCAP Core Loop Started...")

	for range ticker.C {
		m.EvaluateCollisionRisk()
	}
}

func (m *MBCAPCore) changeState(newState domain.MBCAPState) {
	m.currentState = newState
	m.stateTime = time.Now().UnixNano()
	log.Printf("MBCAP State changed: %v", newState)

	// Issue specific MAVLink commands based on state transition
	if newState == domain.STAND_STILL {
		m.broker.Publish(m.config.CmdPublishTopic, map[string]interface{}{"endpoint": "SetFlightmode", "flightmode": "BRAKE"})
	} else if newState == domain.NORMAL {
		m.broker.Publish(m.config.CmdPublishTopic, map[string]interface{}{"endpoint": "SetFlightmode", "flightmode": "GUIDED"})
	} else if newState == domain.EMERGENCY_LAND {
		m.broker.Publish(m.config.CmdPublishTopic, map[string]interface{}{"endpoint": "Land"})
	}
}

func (m *MBCAPCore) EvaluateCollisionRisk() {
	selfBeacon := m.store.GetOwnBeacon()
	if selfBeacon == nil {
		return // Do not process if no telemetry
	}

	externalBeacons := m.store.GetExternalBeacons()

	// 1. Timeouts and deadlocks (simplified to prioritize flow)
	if m.currentState != domain.NORMAL && m.currentState != domain.EMERGENCY_LAND {
		elapsed := time.Now().UnixNano() - m.stateTime
		if elapsed > m.params.HoveringTimeout && m.avoidingBeacon == nil {
			m.changeState(domain.NORMAL)
		}
	}

	// 2. Risk analysis
	if m.currentState == domain.NORMAL {
		var highestPriorityRisk *domain.Beacon

		for _, b := range externalBeacons {
			riskyLocation := hasCollisionRisk(*selfBeacon, b, m.params)
			if riskyLocation != nil && !b.IsLanding {
				// We prioritize UAVs by lower ID just like ArduSim MBCAP
				if highestPriorityRisk == nil || b.UavID < highestPriorityRisk.UavID {
					copy := b
					highestPriorityRisk = &copy
				}
			}
		}

		if highestPriorityRisk != nil {
			m.avoidingBeacon = highestPriorityRisk
			m.idAvoiding = highestPriorityRisk.UavID
			m.changeState(domain.STAND_STILL)
		}
	} else if m.currentState == domain.STAND_STILL {
		if time.Now().UnixNano()-m.stateTime >= m.params.HoveringTimeout {
			if m.avoidingBeacon != nil && m.avoidingBeacon.IsLanding {
				m.avoidingBeacon = nil
				m.idAvoiding = domain.ID_AVOIDING_DEFAULT
				m.changeState(domain.NORMAL)
			} else if m.avoidingBeacon != nil {
				// Passing priority
				if selfBeacon.UavID > m.avoidingBeacon.UavID && m.avoidingBeacon.State == domain.GO_ON_PLEASE {
					m.changeState(domain.OVERTAKING)
				} else if selfBeacon.UavID < m.avoidingBeacon.UavID {
					// We need to move aside or go on please
					log.Println("UAV priority inferior, moving aside...")
					m.changeState(domain.MOVING_ASIDE)
					// In ArduSim, it calculated `needsToMoveAside` and triggered `MoveToPosition`
					// Publishing an evasive maneuver (generic right sidestep for demo)
					m.broker.Publish(m.config.CmdPublishTopic, map[string]interface{}{
						"endpoint":  "MoveToPosition",
						"latitude":  selfBeacon.Points[0].X + 1e-5, // Slightly offset
						"longitude": selfBeacon.Points[0].Y + 1e-5,
						"altitude":  selfBeacon.Points[0].Z,
					})
				}
			}
		}
	} else if m.currentState == domain.MOVING_ASIDE {
		// Once finished moving aside, grant permission
		m.changeState(domain.GO_ON_PLEASE)
	} else if m.currentState == domain.GO_ON_PLEASE {
		// Wait for the other UAV to overtake
		if m.avoidingBeacon != nil && m.avoidingBeacon.Event > selfBeacon.Event {
			// Solved
			m.avoidingBeacon = nil
			m.idAvoiding = domain.ID_AVOIDING_DEFAULT
			m.changeState(domain.NORMAL)
		}
	} else if m.currentState == domain.OVERTAKING {
		elapsed := time.Now().UnixNano() - m.stateTime
		if elapsed > m.params.OvertakeDelayTimeout {
			// Finished overtaking
			selfBeacon.Event++
			m.avoidingBeacon = nil
			m.idAvoiding = domain.ID_AVOIDING_DEFAULT
			m.changeState(domain.NORMAL)
		}
	}

	// Update self beacon to external memory
	selfBeacon.State = m.currentState
	selfBeacon.IdAvoiding = m.idAvoiding
	m.store.UpdateOwnBeacon(*selfBeacon)
}
