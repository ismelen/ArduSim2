package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"application/infrastructure"
	"application/ports"
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

	for {
		if err := broker.Connect(config.BrokerIP, config.BrokerPort, []string{
			config.SuggestionsTopic,
			config.GlobalCommands,
			config.TelemetryTopic,
		}); err != nil {
			log.Printf("Failed to connect to Broker: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	// ---------------------------
	// Inject Broker into Logger
	// ---------------------------
	setupLogger(broker, config.LogsTopic)

	var uavLink *infrastructure.DirectUAVLink
	for {
		uavLink, err = infrastructure.NewDirectUAVLink(config.UAVControllerIP, config.UAVControllerPort, config.UAVTelemetryPort)
		if err != nil {
			log.Printf("Failed to establish direct UDP link to UAV: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	defer uavLink.Close()

	// Core Logic Usecase
	mixer := usecase.NewMovementMixer(broker, uavLink, config)

	// Block and Run Loop
	mixer.Run()
}

func setupLogger(broker ports.Broker, logsTopic string) {
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

	// Add our custom BrokerLogWriter if broker is connected
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
