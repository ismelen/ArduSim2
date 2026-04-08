package usecase

import (
	"fmt"
	"log"

	"external_comms/domain"
	"external_comms/ports"
)

type GatewayBridge struct {
	config  *domain.AppConfig
	broker  ports.Broker
	netLink ports.NetSimLink
}

func NewGatewayBridge(c *domain.AppConfig, b ports.Broker, n ports.NetSimLink) *GatewayBridge {
	return &GatewayBridge{
		config:  c,
		broker:  b,
		netLink: n,
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
	switch msg.Topic {
	case g.config.SubTelemetryTopic:
		// Route internal telemetry to swarm
		extMsg := domain.SendedNetSimMessage{
			Topic:    "telemetry",
			Source:  fmt.Sprintf("%d", g.config.UAVId),
			Payload: msg.Payload,
		}
		g.netLink.Send(extMsg)
		// log.Printf("[Internal->External] Forwarded Telemetry")

	case g.config.SubMessagesTopic:
		// Route internal P2P message to swarm
		extMsg := domain.SendedNetSimMessage{
			Topic:    "broadcast",
			Source:  fmt.Sprintf("%d", g.config.UAVId),
			Payload: msg.Payload,
		}
		g.netLink.Send(extMsg)
		log.Printf("[Internal->External] Forwarded Message")
	}
}

func (g *GatewayBridge) handleExternalNetMessage(msg domain.ReceivedNetSimMessage) {
	// External to Internal
	// Ignore our own echo if NetSim broadcasts everything back
	log.Printf("[External->Internal] Forwarding message from %s: %v", msg.Source, msg.Payload)
	if msg.Source == fmt.Sprintf("%d", g.config.UAVId) {
		return
	}

	g.broker.Publish(msg.Payload)
}
