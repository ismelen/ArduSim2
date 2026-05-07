package ports

import (
	"context"
	"ui/internal/domain"
)

// SimulationUseCase defines the business actions related to the simulation lifecycle.
type SimulationUseCase interface {
	StartSimulation(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error
	StopSimulation()
	SendAlgorithmCommand(serviceId string, command string) error
	LoadSimulationConfig(ctx context.Context) (*domain.SimulationState, error)
	SaveSimulationConfig(uavs []domain.UAV, config domain.GeneralConfig, mode string) error
	DiscardCurrentRun(config domain.GeneralConfig) error
	SelectFile(ctx context.Context) (string, error)
	GetKmlFirstCoordinate(path string) (*domain.Coordinate, error)
	LoadLogEntry(runDir string) (map[string]any, error)
	LoadLogEntries(ctx context.Context) ([]string, error)
	LoadFile(path string) (string, error)
}
