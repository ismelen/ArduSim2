package main

import (
	"fmt"
	"log"
	"os"
	"safe_takeoff/infrastructure/broker"
	"safe_takeoff/infrastructure/config"
	"safe_takeoff/usecase"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <config.json>\n", os.Args[0])
	}

	configFile := os.Args[1]

	// Dependency Injection Setup
	fileLoader := config.NewFileLoader()
	appConfig, err := fileLoader.LoadAppConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	udpBroker := broker.NewUDPBroker()
	brokerAddr := fmt.Sprintf("%s:%d", appConfig.BrokerIP, appConfig.BrokerPort)
	if err := udpBroker.Connect(brokerAddr); err != nil {
		log.Fatalf("Failed to connect to UDP broker: %v", err)
	}
	defer udpBroker.Close()

	// Initialize and Run the Service
	manager := usecase.NewTakeOffManager(udpBroker, appConfig)
	manager.Start()
}
