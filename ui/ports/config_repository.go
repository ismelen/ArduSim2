package ports

import "ui/domain"

// ConfigRepository handles persistence of simulation data and service discovery.
type ConfigRepository interface {
	GetAvailableServices() []domain.ServiceType
	SaveSimulation(simDir string, state domain.SimulationState) error
	LoadSimulation(stateFile string) (*domain.SimulationState, error)
	GetSimulationsDir() string
}
