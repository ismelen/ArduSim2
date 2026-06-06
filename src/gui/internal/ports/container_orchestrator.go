package ports

import "ui/internal/domain"

// ContainerOrchestrator handles the lifecycle of simulated containers.
type ContainerOrchestrator interface {
	Run(swarms []domain.Swarm, config domain.GeneralConfig, isLocal bool, simDir string) (string, error)
	PrepareExport(swarms []domain.Swarm, config domain.GeneralConfig, simDir string) error
	BuildCompose(composePath string) error
	BuildAllImages(simDir string) error
	StartCompose(composePath string) error
	StopCompose(composePath string) error
	StartStack(composePath, swarmHost, stackName string) error
	StopStack(stackName, swarmHost string) error
	CollectSwarmLogs(stackName, swarmHost, simName, destDir string) error
}
