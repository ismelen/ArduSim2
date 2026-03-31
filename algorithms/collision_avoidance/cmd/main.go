package main

import (
	"log"
	"os"
	"time"

	"collision_avoidance/domain"
	"collision_avoidance/infrastructure"
	"collision_avoidance/usecase"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <config.json> <mbcap.properties>\n", os.Args[0])
	}

	configFile := os.Args[1]
	propertiesFile := os.Args[2]

	// Dependency Injection
	fileLoader := infrastructure.NewFileLoader()
	config, err := fileLoader.LoadAppConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	params, err := fileLoader.LoadMBCAPParams(propertiesFile)
	if err != nil {
		log.Fatalf("Failed to load MBCAP properties: %v", err)
	}

	udpBroker := infrastructure.NewUDPBroker()
	defer udpBroker.Close()

	// 3s expiration time for external beacons
	memoryStore := infrastructure.NewMemoryTelemetryStore(params.BeaconExpirationTime)

	// Combine topics to subscribe to
	subTopics := []string{config.TelemetryTopic, config.ExternalTelemetry}
	if err := udpBroker.Connect(config.BrokerIP, config.BrokerPort, subTopics); err != nil {
		log.Fatalf("Failed to connect to broker: %v", err)
	}

	mbcapCore := usecase.NewMBCAPCore(udpBroker, memoryStore, config, params)

	msgChan, err := udpBroker.Listen()
	if err != nil {
		log.Fatalf("Expected listener to start: %v", err)
	}

	// Background routine listening to Telemetry and translating into Beacons
	go func() {
		for msg := range msgChan {
			if msg.Topic == config.TelemetryTopic || msg.Topic == config.ExternalTelemetry {
				// Parse simple JSON telemetry into Beacon
				uavidFloat, _ := msg.Payload["uav_id"].(float64)
				lat, _ := msg.Payload["lat"].(float64)
				lon, _ := msg.Payload["lon"].(float64)
				alt, _ := msg.Payload["relative_alt"].(float64)
				speed, _ := msg.Payload["speed"].(float64) // mock value

				// Basic points prediction mimicking ArduSim geometry
				var points []domain.Location3DUTM
				for i := 0; i < 5; i++ {
					points = append(points, domain.Location3DUTM{
						X: lon + speed*float64(i)*0.0001, // arbitrary dummy progression
						Y: lat + speed*float64(i)*0.0001,
						Z: alt,
					})
				}

				beacon := domain.Beacon{
					UavID:      int64(uavidFloat),
					State:      domain.NORMAL,
					IsLanding:  false,
					Speed:      speed,
					Time:       time.Now().UnixNano(),
					Points:     points,
					Event:      0,
					IdAvoiding: domain.ID_AVOIDING_DEFAULT,
				}

				if msg.Topic == config.TelemetryTopic {
					beacon.UavID = config.UavID       // ensure ID is correct for self
					mbcapCore.EvaluateCollisionRisk() // Call core loop
					memoryStore.UpdateOwnBeacon(beacon)
				} else {
					memoryStore.UpdateExternalBeacon(beacon)
				}
			}
		}
	}()

	mbcapCore.Run() // Blocks and runs the ticking loop
}
