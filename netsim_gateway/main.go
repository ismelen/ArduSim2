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

	uavConn := udp.NewConnection(cfg.UAVListenPort, log)
	defer uavConn.Close()

	netsimConn := udp.NewConnection(cfg.NetsimListenPort, log)
	defer netsimConn.Close()

	uavSender := udp.NewSender(uavConn, log)
	netsimSender := udp.NewSender(netsimConn, log)
	uavRecv := udp.NewReceiver(uavConn, log)
	netsimRecv := udp.NewReceiver(netsimConn, log)

	netsimAddrs := config.DiscoverNetsims(cfg.NetsimDiscovery)
	gateway := usecase.NewGateway(usecase.Config{
		UAVListenPort:     cfg.UAVListenPort,
		NetsimListenPort:  cfg.NetsimListenPort,
		SnapshotIntervalS: cfg.SnapshotIntervalS,
		NetsimDiscovery:   cfg.NetsimDiscovery,
	}, netsimAddrs, uavSender, netsimSender, log)

	uavHandler := usecase.NewUAVHandler(gateway)
	netsimHandler := usecase.NewNetsimHandler(gateway)

	go uavRecv.Run(uavHandler)
	go netsimRecv.Run(netsimHandler)
	go uavSender.Run()
	go netsimSender.Run()
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
