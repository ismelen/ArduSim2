package usecase

import (
	"log"

	"external_comms/domain"
	"external_comms/ports"
)

type GatewayBridge struct {
	config  *domain.AppConfig
	broker  ports.Broker
	netLink ports.NetSimLink
	uavID   string
}

func NewGatewayBridge(c *domain.AppConfig, b ports.Broker, n ports.NetSimLink, uavID string) *GatewayBridge {
	return &GatewayBridge{
		config:  c,
		broker:  b,
		netLink: n,
		uavID:   uavID,
	}
}

func (g *GatewayBridge) Run() {
	log.Println("Gateway Bridge AUX N Started...")

	brokerChan, err := g.broker.Listen()
	if err != nil {
		log.Fatalf("Broker listen failed: %v", err)
	}

	netChan, err := g.netLink.Listen()
	if err != nil {
		log.Fatalf("NetSim listen failed: %v", err)
	}

	for {
		select {
		case bMsg := <-brokerChan:
			g.handleInternalBrokerMessage(bMsg)
		case nMsg := <-netChan:
			g.handleExternalNetMessage(nMsg)
		}
	}
}

func (g *GatewayBridge) handleInternalBrokerMessage(msg ports.BrokerMessage) {
	// Internal to External
	if msg.Topic == g.config.SubTelemetryTopic {
		// Route internal telemetry to swarm
		extMsg := domain.NetSimMessage{
			Type:    "telemetry",
			Source:  g.uavID,
			Payload: msg.Payload,
		}
		g.netLink.Send(extMsg)
		log.Printf("[Internal->External] Forwarded Telemetry")

	} else if msg.Topic == g.config.SubMessagesTopic {
		// Route internal P2P message to swarm
		extMsg := domain.NetSimMessage{
			Type:    "message",
			Source:  g.uavID,
			Payload: msg.Payload,
		}
		g.netLink.Send(extMsg)
		log.Printf("[Internal->External] Forwarded Message")
	}
}

func (g *GatewayBridge) handleExternalNetMessage(msg domain.NetSimMessage) {
	// External to Internal
	// Ignore our own echo if NetSim broadcasts everything back
	if msg.Source == g.uavID {
		return
	}

	if msg.Type == "telemetry" {
		g.broker.Publish(g.config.PubExternalTelemetryTopic, msg.Payload)
		log.Printf("[External->Internal] Received Swarm Telemetry from %s", msg.Source)
	} else if msg.Type == "message" {
		g.broker.Publish(g.config.PubExternalMessagesTopic, msg.Payload)
		log.Printf("[External->Internal] Received Swarm Message from %s", msg.Source)
	}
}
