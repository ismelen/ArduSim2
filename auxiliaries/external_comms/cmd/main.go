package main

import (
	"log"
	"os"

	"external_comms/infrastructure"
	"external_comms/usecase"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <config.json> <uav_id>\n", os.Args[0])
	}
	configFile := os.Args[1]
	uavID := os.Args[2]

	fileLoader := infrastructure.NewFileLoader()
	config, err := fileLoader.LoadAppConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	broker := infrastructure.NewUDPBroker()
	defer broker.Close()

	if err := broker.Connect(config.BrokerIP, config.BrokerPort, []string{
		config.SubTelemetryTopic,
		config.SubMessagesTopic,
	}); err != nil {
		log.Fatalf("Failed to connect broker: %v", err)
	}

	netLink, err := infrastructure.NewUDPNetSimLink(config.NetSimIP, config.NetSimPort)
	if err != nil {
		log.Fatalf("Failed to connect NetSim: %v", err)
	}
	defer netLink.Close()

	bridge := usecase.NewGatewayBridge(config, broker, netLink, uavID)
	bridge.Run()
}
