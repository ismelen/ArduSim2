package main

import (
	"log"
	"os"

	"application/infrastructure"
	"application/usecase"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <config.json>\n", os.Args[0])
	}

	configFile := os.Args[1]

	// Dependency Injection Loaders
	fileLoader := infrastructure.NewFileLoader()
	config, err := fileLoader.LoadAppConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	if config.MixWindowMs == 0 {
		config.MixWindowMs = 200 // Default to 5Hz sampling
	}

	// Infra
	broker := infrastructure.NewUDPBroker()
	defer broker.Close()

	if err := broker.Connect(config.BrokerIP, config.BrokerPort, []string{
		config.SuggestionsTopic,
		config.GlobalCommands,
		config.TelemetryTopic,
	}); err != nil {
		log.Fatalf("Failed to connect to Broker: %v", err)
	}

	uavLink, err := infrastructure.NewDirectUAVLink(config.UAVControllerIP, config.UAVControllerPort, config.UAVTelemetryPort)
	if err != nil {
		log.Fatalf("Failed to establish direct UDP link to UAV: %v", err)
	}
	defer uavLink.Close()

	// Core Logic Usecase
	mixer := usecase.NewMovementMixer(broker, uavLink, config)

	// Block and Run Loop
	mixer.Run()
}
