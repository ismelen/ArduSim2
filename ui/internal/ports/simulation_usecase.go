package ports

import (
	"context"
	"ui/internal/domain"
)

// SimulationUseCase defines the business actions related to the simulation lifecycle.
type SimulationUseCase interface {
	StartSimulation(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error
	DownloadLogs() error
	StopSimulation()
	SendAlgorithmCommand(serviceId string, command string) error
	LoadSimulationConfig(ctx context.Context) (*domain.SimulationState, error)
	SaveSimulationConfig(uavs []domain.UAV, config domain.GeneralConfig, mode string) error
	DiscardCurrentRun(config domain.GeneralConfig) error
	SelectFile(ctx context.Context) (string, error)
	SelectSpeedProfile(ctx context.Context) (string, error)
	GetKmlFirstCoordinate(path string) (*domain.Coordinate, error)
	LoadLogEntries(ctx context.Context) ([]string, error)
	SearchLogs(zipPath string, filter domain.LogFilter) ([]domain.LogMessage, error)
	LoadFile(path string) (string, error)
}
