package usecases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui/internal/domain"
	"ui/internal/infrastructure/util"
	"ui/internal/ports"
)

type SimulationInteractor struct {
	orchestrator ports.ContainerOrchestrator
	subscriber   ports.TelemetrySubscriber
	repo         ports.ConfigRepository
	ui           ports.UIBridge
	logger       ports.LoggerClient
	session      *activeSession
}

func NewSimulationInteractor(
	orchestrator ports.ContainerOrchestrator,
	subscriber ports.TelemetrySubscriber,
	repo ports.ConfigRepository,
	ui ports.UIBridge,
	logger ports.LoggerClient,
) *SimulationInteractor {
	return &SimulationInteractor{
		orchestrator: orchestrator,
		subscriber:   subscriber,
		repo:         repo,
		ui:           ui,
		logger:       logger,
		session:      newActiveSession(),
	}
}

func (i *SimulationInteractor) StartSimulation(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error {
	config.SanitizeSimulationName()
	simDir := filepath.Join(i.repo.GetSimulationsDir(), config.SimulationName)

	composePath, err := i.orchestrator.Run(uavs, config, isLocal, simDir)
	if err != nil {
		return fmt.Errorf("prepare simulation: %w", err)
	}

	i.session.clear()
	i.session.ctx = ctx
	i.session.composePath = composePath
	i.session.simulationName = config.SimulationName
	i.session.uavIDs = make([]string, len(uavs))
	for idx, uav := range uavs {
		i.session.uavIDs[idx] = uav.ID
	}
	i.session.loggingEnabled = config.LoggingEnabled

	if isLocal {
		if err := i.orchestrator.StartCompose(composePath); err != nil {
			return err
		}

		i.subscriber.SetExpectedFleet(i.session.uavIDs)

		// Algorithm tracking
		algoIDs := util.CollectAlgorithmIDs(uavs)
		for _, id := range algoIDs {
			i.session.algorithmIDs[id] = true
		}

		i.subscriber.SetOnFinish(func() {
			for _, algo := range algoIDs {
				if err := i.SendAlgorithmCommand(algo, "stop"); err != nil {
					fmt.Printf("[interactor] failed to stop algorithm %s: %v\n", algo, err)
				}
			}
		})
	} else {
		// SWARM MODE
		stackName := "Ardusim2-" + config.SimulationName
		i.session.stackName = stackName
		i.session.swarmHost = config.SwarmHost

		if err := i.orchestrator.StartStack(composePath, config.SwarmHost, stackName); err != nil {
			return err
		}

		swarmIP, _, _ := strings.Cut(config.SwarmHost, ":")
		i.subscriber.SetRemoteAddr(swarmIP)
		i.subscriber.SetExpectedFleet(i.session.uavIDs)
	}
	go i.subscriber.Start(ctx)

	return nil
}

func (i *SimulationInteractor) BuildImages(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error {
	config.SanitizeSimulationName()
	simDir := filepath.Join(i.repo.GetSimulationsDir(), config.SimulationName)

	// Build all known images and all local algorithms, independent of simulation size/mode
	return i.orchestrator.BuildAllImages(simDir)
}


func (i *SimulationInteractor) StopSimulation() {
	if i.session.stackName != "" {
		_ = i.orchestrator.CollectSwarmLogs(i.session.stackName, i.session.swarmHost, i.session.simulationName, "")
		_ = i.orchestrator.StopStack(i.session.stackName, i.session.swarmHost)
	} else if i.session.composePath != "" {
		_ = i.orchestrator.StopCompose(i.session.composePath)
	}
	i.session.clear()
}

func (i *SimulationInteractor) SendAlgorithmCommand(serviceId string, command string) error {
	payload := map[string]interface{}{
		"topic":   "algo/" + serviceId,
		"payload": map[string]interface{}{
			"command": command,
		},
	}
	if err := i.subscriber.SendGlobalBroadcast(payload); err != nil {
		return err
	}

	i.ui.EmitEvent("netsim:message", map[string]string{
		"source":  "[Global] ",
		"label":   fmt.Sprintf("[Global] Command '%s' sent to service: %s", command, serviceId),
	})

	if command == "stop" && len(i.session.algorithmIDs) > 0 {
		if i.session.markStopped(serviceId) {
			i.subscriber.NotifyUserStoppedAll()
		}
	}
	return nil
}

func (i *SimulationInteractor) DownloadLogs() error {
	if !i.session.isRunning() && i.session.simulationName == "" {
		return fmt.Errorf("no active simulation")
	}
	simDir := filepath.Join(i.repo.GetSimulationsDir(), i.session.simulationName)
	logsDir := filepath.Join(simDir, "logs")
	os.MkdirAll(logsDir, 0755)

	loggerZipBytes, err := i.logger.DownloadZip(i.session.loggerHost())
	if err != nil {
		return fmt.Errorf("fetch logger zip: %w", err)
	}

	uavLogDirs := map[string]string{}
	if i.session.loggingEnabled {
		for _, uavID := range i.session.uavIDs {
			dir := filepath.Join(simDir, "uav_logs", uavID)
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				uavLogDirs[uavID] = dir
			}
		}
	}

	timestamp := time.Now().Format("20060102_150405")
	outPath := filepath.Join(logsDir, fmt.Sprintf("logs_%s.zip", timestamp))
	return buildCombinedZip(loggerZipBytes, uavLogDirs, outPath)
}
