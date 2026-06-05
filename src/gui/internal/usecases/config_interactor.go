package usecases

import (
	"context"
	"os"
	"path/filepath"

	"ui/internal/domain"
	"ui/internal/infrastructure/kml"
	"ui/internal/ports"
)

type ConfigInteractor struct {
	repo ports.ConfigRepository
	ui   ports.UIBridge
}

func NewConfigInteractor(repo ports.ConfigRepository, ui ports.UIBridge) *ConfigInteractor {
	return &ConfigInteractor{
		repo: repo,
		ui:   ui,
	}
}

func (i *ConfigInteractor) LoadFile(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	return string(bytes), err
}

func (i *ConfigInteractor) LoadSimulationConfig(ctx context.Context) (*domain.SimulationState, error) {
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

func (i *ConfigInteractor) SaveSimulationConfig(uavs []domain.UAV, config domain.GeneralConfig, mode string) error {
	config.SanitizeSimulationName()
	simDir := filepath.Join(i.repo.GetSimulationsDir(), config.SimulationName)

	state := domain.SimulationState{
		UAVs:          uavs,
		GeneralConfig: config,
		ActiveMode:    mode,
	}

	return i.repo.SaveSimulation(simDir, state)
}

func (i *ConfigInteractor) DiscardCurrentRun(config domain.GeneralConfig) error {
	config.SanitizeSimulationName()
	// This logic remains similar, but using the repo
	return nil // Implementation detail for later
}

func (i *ConfigInteractor) SelectFile(ctx context.Context) (string, error) {
	return i.ui.OpenFileDialog(ctx, "Select Auxiliary File", []ports.FileFilter{
		{DisplayName: "KML files (*.kml)", Pattern: "*.kml"},
		{DisplayName: "All files", Pattern: "*.*"},
	})
}

func (i *ConfigInteractor) SelectArduPilotInstance(ctx context.Context) (string, error) {
	return i.ui.OpenFileDialog(ctx, "Select ArduPilot Instance", []ports.FileFilter{
		{DisplayName: "Executables (*)", Pattern: "*"},
		{DisplayName: "All files", Pattern: "*.*"},
	})
}

func (i *ConfigInteractor) GetKmlFirstCoordinate(path string) (*domain.Coordinate, error) {
	return kml.GetFirstCoordinate(path)
}
