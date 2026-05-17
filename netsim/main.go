package main

import (
	"netsim/domain/service"
	"netsim/infra/config"
	"netsim/infra/logger"
	"netsim/infra/udp"
	"netsim/usecase"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.LoadConfig("config.json")
	log := logger.NewConsoleLogger(cfg.Log.Level)

	conn := udp.NewConnection(cfg.ListenPort, log)
	defer conn.Close()
	
	sender := udp.NewSender(conn, cfg.GatewayAddr, log)
	receiver := udp.NewReceiver(conn, log)

	nodeId := os.Getenv("NODE_ID")
	spatial := service.NewSpatialGrid(cfg.Simulation.ChunkSizeM)
	sim := usecase.NewSimulator(nodeId, cfg.Simulation, spatial, sender, log)

	go receiver.Run(sim)
	go sender.Run()
	go usecase.RunSnapshotEmitter(sim, sender, cfg.Simulation.SnapshotIntervalS, log)

	go func() {
		ticker := time.NewTicker(time.Millisecond * 1)
		for range ticker.C {
			sim.SendMessages()
		}
	}()

	log.Info("Netsim started", "node_id", nodeId, "port", cfg.ListenPort)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down Netsim...")
}
