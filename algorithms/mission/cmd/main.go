package main

import (
	"log"
	"os"

	"mission/infrastructure/broker"
	"mission/infrastructure/config"
	"mission/infrastructure/parser"
	"mission/usecase"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <config.json>\n", os.Args[0])
	}

	configFile := os.Args[1]

	// Dependency Injection
	udpBroker := broker.NewUDPBroker()
	defer udpBroker.Close()

	fileLoader := config.NewFileLoader()
	kmlParser := parser.NewKmlParser()

	manager := usecase.NewMissionManager(udpBroker, fileLoader, kmlParser)

	if err := manager.Initialize(configFile); err != nil {
		log.Fatalf("Failed to initialize mission manager: %v", err)
	}

	manager.Run()
}
