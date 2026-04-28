package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"application/infrastructure"
	"application/usecase"
)

func main() {
	setupLogger()

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
		panic(err)
	}

	uavLink, err := infrastructure.NewDirectUAVLink(config.UAVControllerIP, config.UAVControllerPort, config.UAVTelemetryPort)
	if err != nil {
		log.Fatalf("Failed to establish direct UDP link to UAV: %v", err)
		panic(err)
	}
	defer uavLink.Close()

	// Core Logic Usecase
	mixer := usecase.NewMovementMixer(broker, uavLink, config)

	// Block and Run Loop
	mixer.Run()
}

func setupLogger() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	outputs := []io.Writer{os.Stdout}

	// If /app/logs exists, add a file writer
	logDir := "/app/logs"
	if info, err := os.Stat(logDir); err == nil && info.IsDir() {
		logFile, err := os.OpenFile(filepath.Join(logDir, "application.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err == nil {
			outputs = append(outputs, logFile)
			fmt.Printf("Logging to %s/application.log\n", logDir)
		} else {
			fmt.Printf("Warning: failed to open log file: %v\n", err)
		}
	}

	multi := io.MultiWriter(outputs...)
	log.SetOutput(multi)

	if os.Getenv("DEBUG") == "true" {
		log.Println("Verbose logging enabled (DEBUG=true)")
	}
}
