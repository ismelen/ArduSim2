package usecases

import (
	"context"
	"fmt"
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

	// State for the active simulation session
	activeComposePath    string
	activeStackName       string
	activeSwarmHost       string
	activeSimulationName  string
	activeAlgorithmIDs    map[string]bool
	stoppedAlgorithmIDs   map[string]bool
}

func NewSimulationInteractor(
	orchestrator ports.ContainerOrchestrator,
	subscriber ports.TelemetrySubscriber,
	repo ports.ConfigRepository,
	ui ports.UIBridge,
) *SimulationInteractor {
	return &SimulationInteractor{
		orchestrator:        orchestrator,
		subscriber:          subscriber,
		repo:                repo,
		ui:                  ui,
		activeAlgorithmIDs:  make(map[string]bool),
		stoppedAlgorithmIDs: make(map[string]bool),
	}
}

func (i *SimulationInteractor) StartSimulation(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, mode string, isLocal bool) error {
	config.SanitizeSimulationName()
	simDir := filepath.Join(i.repo.GetSimulationsDir(), config.SimulationName)

	composePath, err := i.orchestrator.Run(uavs, config, mode, isLocal, simDir)
	if err != nil {
		return fmt.Errorf("prepare simulation: %w", err)
	}

	i.activeComposePath = composePath
	i.activeStackName = ""
	i.activeSwarmHost = ""
	i.activeSimulationName = config.SimulationName

	if isLocal {
		if err := i.orchestrator.StartCompose(composePath); err != nil {
			return err
		}

		uavIDs := make([]string, len(uavs))
		for idx, uav := range uavs {
			uavIDs[idx] = uav.ID
		}
		i.subscriber.SetExpectedFleet(uavIDs)

		// Algorithm tracking
		algoIDs := util.CollectAlgorithmIDs(uavs)
		i.activeAlgorithmIDs = make(map[string]bool)
		i.stoppedAlgorithmIDs = make(map[string]bool)
		for _, id := range algoIDs {
			i.activeAlgorithmIDs[id] = true
		}

		i.subscriber.SetOnFinish(func() {
			for _, algo := range algoIDs {
				if err := i.SendAlgorithmCommand(algo, "stop"); err != nil {
					fmt.Printf("[interactor] failed to stop algorithm %s: %v\n", algo, err)
				}
			}
		})

		go i.subscriber.Start(ctx)
	} else {
		// SWARM MODE
		stackName := "Ardusim2-" + config.SimulationName
		i.activeStackName = stackName
		i.activeSwarmHost = config.SwarmHost

		if err := i.orchestrator.StartStack(composePath, config.SwarmHost, stackName); err != nil {
			return err
		}

		swarmIP, _, _ := strings.Cut(config.SwarmHost, ":")
		i.subscriber.SetRemoteAddr(swarmIP)

		uavIDs := make([]string, len(uavs))
		for idx, uav := range uavs {
			uavIDs[idx] = uav.ID
		}
		i.subscriber.SetExpectedFleet(uavIDs)

		go i.subscriber.Start(ctx)
	}

	return nil
}

func (i *SimulationInteractor) StopSimulation() {
	if i.activeStackName != "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		destDir := filepath.Join(i.repo.GetSimulationsDir(), i.activeSimulationName, "runs", timestamp)
		
		_ = i.orchestrator.CollectSwarmLogs(i.activeStackName, i.activeSwarmHost, i.activeSimulationName, destDir)
		_ = i.orchestrator.StopStack(i.activeStackName, i.activeSwarmHost)
		
		i.activeStackName = ""
		i.activeSwarmHost = ""
		i.activeSimulationName = ""
	} else if i.activeComposePath != "" {
		_ = i.orchestrator.StopCompose(i.activeComposePath)
		i.activeComposePath = ""
	}
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

	if command == "stop" && len(i.activeAlgorithmIDs) > 0 {
		i.stoppedAlgorithmIDs[serviceId] = true
		allStopped := true
		for id := range i.activeAlgorithmIDs {
			if !i.stoppedAlgorithmIDs[id] {
				allStopped = false
				break
			}
		}
		if allStopped {
			// Notify subscriber via context? Actually app.go used a.ctx
			// We should probably pass a context or use one from start
			i.subscriber.NotifyUserStoppedAll(context.Background()) // Simplified for now
		}
	}
	return nil
}

func (i *SimulationInteractor) LoadSimulationConfig(ctx context.Context) (*domain.SimulationState, error) {
	selectedDir, err := i.ui.OpenDirectoryDialog(ctx, "Select Simulation Directory", i.repo.GetSimulationsDir())
	if err != nil || selectedDir == "" {
		return nil, err
	}

	stateFile := filepath.Join(selectedDir, "simulation.json")
	state, err := i.repo.LoadSimulation(stateFile)
	if err != nil {
		return nil, err
	}

	// Repair logic
	availableServices := i.repo.GetAvailableServices()
	svcMap := make(map[string]string)
	for _, s := range availableServices {
		svcMap[s.ID] = s.FolderName
	}

	for idxUAV := range state.UAVs {
		for idxSvc := range state.UAVs[idxUAV].Services {
			svc := &state.UAVs[idxUAV].Services[idxSvc]
			if svc.FolderName == "" {
				if folder, ok := svcMap[svc.ServiceId]; ok {
					svc.FolderName = folder
				} else {
					svc.FolderName = svc.ServiceId
				}
			}
		}
	}

	state.GeneralConfig.OriginalSimulationName = filepath.Base(selectedDir)
	state.GeneralConfig.SimulationName = filepath.Base(selectedDir)

	return state, nil
}

func (i *SimulationInteractor) SaveSimulationConfig(uavs []domain.UAV, config domain.GeneralConfig, mode string) error {
	config.SanitizeSimulationName()
	simDir := filepath.Join(i.repo.GetSimulationsDir(), config.SimulationName)

	state := domain.SimulationState{
		UAVs:          uavs,
		GeneralConfig: config,
		ActiveMode:    mode,
	}

	return i.repo.SaveSimulation(simDir, state)
}

func (i *SimulationInteractor) DiscardCurrentRun(config domain.GeneralConfig) error {
	config.SanitizeSimulationName()
	// This logic remains similar, but using the repo
	return nil // Implementation detail for later
}

func (i *SimulationInteractor) SelectFile(ctx context.Context) (string, error) {
	return i.ui.OpenFileDialog(ctx, "Select Auxiliary File", []ports.FileFilter{
		{DisplayName: "KML files (*.kml)", Pattern: "*.kml"},
		{DisplayName: "All files", Pattern: "*.*"},
	})
}

func (i *SimulationInteractor) GetKmlFirstCoordinate(path string) (*domain.Coordinate, error) {
	// This could be domain logic or a helper.
	return nil, nil // Implementation detail for later
}
