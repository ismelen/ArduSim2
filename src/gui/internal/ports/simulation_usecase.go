package ports

import (
	"context"
	"ui/internal/domain"
)

// SimulationUseCase defines the business actions related to the simulation lifecycle.
type SimulationUseCase interface {
	StartSimulation(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error
	BuildImages(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error
	StopSimulation()
	SendAlgorithmCommand(serviceId string, command string) error
	DownloadLogs() error
}
