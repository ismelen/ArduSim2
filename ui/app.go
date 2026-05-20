package main

import (
	"context"
	"fmt"
	"os"
	"ui/internal/domain"
	"ui/internal/infrastructure/docker"
	"ui/internal/infrastructure/filesystem"
	"ui/internal/infrastructure/netsim"
	"ui/internal/infrastructure/wails"
	"ui/internal/ports"
	"ui/internal/usecases"
)

// App is the Wails binding layer. It delegates operations to use cases.
type App struct {
	ctx           context.Context
	simulation    ports.SimulationUseCase
	discovery     ports.DiscoveryUseCase
	ui            ports.UIBridge
}

// NewApp wires the application using Clean Architecture principles.
func NewApp() *App {
	workDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("cannot determine working directory: %v", err))
	}

	// 1. Infrastructure (Adapters)
	repo := filesystem.NewFileRepository(workDir)
	bridge := wails.NewWailsBridge()
	orchestrator := docker.NewDockerOrchestrator(workDir, bridge)
	subscriber := netsim.NewNetsimSubscriber(bridge)

	// 2. Use Cases (Interactors)
	simUC := usecases.NewSimulationInteractor(orchestrator, subscriber, repo, bridge)
	discUC := usecases.NewDiscoveryInteractor(repo)

	return &App{
		simulation: simUC,
		discovery:  discUC,
		ui:         bridge,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.ui.SetContext(ctx)
}

func (a *App) shutdown() {
	a.simulation.StopSimulation()
}

// --- Wails Bindings ---

func (a *App) GetAvailableServices() []domain.ServiceType {
	return a.discovery.GetAvailableServices()
}

func (a *App) StartSimulation(uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error {
	return a.simulation.StartSimulation(a.ctx, uavs, config, isLocal)
}

func (a *App) StopSimulation() {
	a.simulation.StopSimulation()
}

func (a *App) DownloadLogs() error {
	return a.simulation.DownloadLogs()
}

func (a *App) SendAlgorithmCommand(serviceId string, command string) error {
	return a.simulation.SendAlgorithmCommand(serviceId, command)
}

func (a *App) LoadSimulationConfig() (*domain.SimulationState, error) {
	return a.simulation.LoadSimulationConfig(a.ctx)
}

func (a *App) SaveSimulationConfig(uavs []domain.UAV, config domain.GeneralConfig, mode string) error {
	return a.simulation.SaveSimulationConfig(uavs, config, mode)
}

func (a *App) DiscardCurrentRun(config domain.GeneralConfig) error {
	return a.simulation.DiscardCurrentRun(config)
}

func (a *App) SelectFile() (string, error) {
	return a.simulation.SelectFile(a.ctx)
}

func (a *App) GetKmlFirstCoordinate(path string) (*domain.Coordinate, error) {
	return a.simulation.GetKmlFirstCoordinate(path)
}

func (a *App) LoadLogEntries() ([]string, error) {
	return a.simulation.LoadLogEntries(a.ctx)
}

func (a *App) LoadLogEntry(path string) (map[string]any, error) {
	return a.simulation.LoadLogEntry(path)
}

func (a *App) LoadFile(path string) (string, error) {
	return a.simulation.LoadFile(path)
}
