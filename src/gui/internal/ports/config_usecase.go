package ports

import (
	"context"
	"ui/internal/domain"
)

type ConfigUseCase interface {
	LoadSimulationConfig(ctx context.Context) (*domain.SimulationState, error)
	SaveSimulationConfig(uavs []domain.UAV, config domain.GeneralConfig, mode string) error
	DiscardCurrentRun(config domain.GeneralConfig) error
	SelectFile(ctx context.Context) (string, error)
	SelectSpeedProfile(ctx context.Context) (string, error)
	GetKmlFirstCoordinate(path string) (*domain.Coordinate, error)
	LoadFile(path string) (string, error)
}
