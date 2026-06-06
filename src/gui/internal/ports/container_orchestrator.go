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
	StartKubernetes(manifestPath, dockerHubUser string) error
	StopKubernetes(simName string) error
	CollectKubernetesLogs(simName, destDir string) error
}
