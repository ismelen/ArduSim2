package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"external_comms/infrastructure"
	"external_comms/usecase"
)

func main() {
	setupLogger()

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <config.json>\n", os.Args[0])
	}
	configFile := os.Args[1]

	fileLoader := infrastructure.NewFileLoader()
	config, err := fileLoader.LoadAppConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	broker := infrastructure.NewUDPBroker()
	defer broker.Close()

	for {
		if err := broker.Connect(config.BrokerIP, config.BrokerPort, []string{
			config.SubTelemetryTopic,
			config.SubMessagesTopic,
		}); err != nil {
			log.Printf("Failed to connect broker: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	var netLink *infrastructure.UDPNetSimLink
	for {
		netLink, err = infrastructure.NewUDPNetSimLink(config.NetSimIP, config.NetSimPort)
		if err != nil {
			log.Fatalf("Failed to connect NetSim: %v", err)
			time.Sleep(5 *time.Second)
			continue
		}
		break
	}
	defer netLink.Close()

	bridge := usecase.NewGatewayBridge(config, broker, netLink)
	bridge.Run()
}

func setupLogger() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	outputs := []io.Writer{os.Stdout}

	// If /app/logs exists, add a file writer
	logDir := "/app/logs"
	if info, err := os.Stat(logDir); err == nil && info.IsDir() {
		logFile, err := os.OpenFile(filepath.Join(logDir, "external_comms.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err == nil {
			outputs = append(outputs, logFile)
			fmt.Printf("Logging to %s/external_comms.log\n", logDir)
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
