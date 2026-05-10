package main

import (
	"netsim_gateway/infra/config"
	"netsim_gateway/infra/logger"
	"netsim_gateway/infra/udp"
	"netsim_gateway/usecase"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.LoadConfig("config.json")
	log := logger.NewConsoleLogger(cfg.Log.Level)

	sender := udp.NewSender(log)
	uavRecv := udp.NewReceiver(cfg.UAVListenPort, log)
	netsimRecv := udp.NewReceiver(cfg.NetsimListenPort, log)

	netsimAddrs := config.DiscoverNetsims(cfg.NetsimDiscovery)
	gateway := usecase.NewGateway(usecase.Config{
		UAVListenPort:     cfg.UAVListenPort,
		NetsimListenPort:  cfg.NetsimListenPort,
		SnapshotIntervalS: cfg.SnapshotIntervalS,
		NetsimDiscovery:   cfg.NetsimDiscovery,
	}, netsimAddrs, sender, sender, log)

	uavHandler := usecase.NewUAVHandler(gateway)
	netsimHandler := usecase.NewNetsimHandler(gateway)

	go uavRecv.Run(uavHandler)
	go netsimRecv.Run(netsimHandler)
	go sender.Run()
	go usecase.RunAggregatedSnapshotEmitter(gateway)

	if cfg.NetsimDiscovery.Mode == "swarm" {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.NetsimDiscovery.RediscoverIntervalS) * time.Second)
			for range ticker.C {
				newAddrs := config.DiscoverNetsims(cfg.NetsimDiscovery)
				gateway.UpdateNetsims(newAddrs)
			}
		}()
	}

	log.Info("Gateway started", "uav_port", cfg.UAVListenPort, "netsim_port", cfg.NetsimListenPort)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down Gateway...")
}
