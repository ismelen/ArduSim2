package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"mission/infrastructure"
	"mission/infrastructure/broker"
	"mission/infrastructure/config"
	"mission/infrastructure/parser"
	"mission/ports"
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

	setupLogger(udpBroker, manager.GetConfig().LogsTopic)

	manager.Run()
}

func setupLogger(broker ports.Broker, logsTopic string) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	outputs := []io.Writer{os.Stdout}

	// If /app/logs exists, add a file writer
	logDir := "/app/logs"
	if info, err := os.Stat(logDir); err == nil && info.IsDir() {
		logFile, err := os.OpenFile(filepath.Join(logDir, "mission.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err == nil {
			outputs = append(outputs, logFile)
			fmt.Printf("Logging to %s/mission.log\n", logDir)
		} else {
			fmt.Printf("Warning: failed to open log file: %v\n", err)
		}
	}

	if broker != nil && logsTopic != "" {
		brokerWriter := infrastructure.NewBrokerLogWriter(broker, logsTopic)
		outputs = append(outputs, brokerWriter)
	}

	multi := io.MultiWriter(outputs...)
	log.SetOutput(multi)

	if os.Getenv("DEBUG") == "true" {
		log.Println("Verbose logging enabled (DEBUG=true)")
	}
}
