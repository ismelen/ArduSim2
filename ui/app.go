package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"ui/internal/config"
	"ui/internal/netsim"
	"ui/internal/service"
	"ui/internal/simulation"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails binding layer. It holds no business logic — it delegates
// every operation to the appropriate internal package.
type App struct {
	ctx               context.Context
	simCtx            context.Context
	simCancel         context.CancelFunc
	paths             config.Paths
	discoverer        *service.Discoverer
	subscriber        *netsim.Subscriber
	// activeComposePath holds the path of the last generated docker-compose.yaml
	// so that shutdown() can bring down the containers cleanly.
	activeComposePath string
}

// NewApp creates the App, resolving all project paths relative to the
// binary's working directory.
func NewApp() *App {
	workDir, err := os.Getwd()
	if err != nil {
		// Unrecoverable at startup — panic with a clear message.
		panic(fmt.Sprintf("cannot determine working directory: %v", err))
	}

	paths := config.NewPaths(workDir)
	return &App{
		paths:      paths,
		discoverer: service.NewDiscoverer(paths.AlgorithmsDir),
		subscriber: netsim.NewSubscriber(),
	}
}

// startup is called by Wails when the application window is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetAvailableServices returns all algorithm services that expose a schema.json.
func (a *App) GetAvailableServices() []simulation.ServiceType {
	return a.discoverer.GetAvailableServices()
}

// StartSimulation generates the Docker Compose environment for the given fleet
// and, when isLocal is true, launches it and subscribes to network telemetry.
func (a *App) StartSimulation(uavs []simulation.UAV, generalConfig simulation.GeneralConfig, activeMode string, isLocal bool) error {
	orchestrator := simulation.NewOrchestrator(a.paths)

	composePath, err := orchestrator.Run(uavs, generalConfig, activeMode, isLocal)
	if err != nil {
		return fmt.Errorf("prepare simulation: %w", err)
	}

	a.activeComposePath = composePath

	if isLocal {
		if err := launchDockerCompose(composePath); err != nil {
			return err
		}
		
		// Create a session-specific context that can be cancelled without killing the app.
		if a.simCancel != nil {
			a.simCancel()
		}
		a.simCtx, a.simCancel = context.WithCancel(a.ctx)
		go a.subscriber.Start(a.simCtx)
	}

	return nil
}

// LoadSimulationConfig opens a directory picker and attempts to read a
// simulation.json state file from the selected directory.
func (a *App) LoadSimulationConfig() (*simulation.SimulationState, error) {
	selectedDir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: a.paths.SimulationsDir,
		Title:            "Select Simulation Directory",
	})

	if err != nil {
		return nil, fmt.Errorf("open dialog: %w", err)
	}

	if selectedDir == "" {
		return nil, nil // Cancelled by user
	}

	stateFile := filepath.Join(selectedDir, "simulation.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("read simulation.json: %w", err)
	}

	var state simulation.SimulationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse simulation state: %w", err)
	}

	return &state, nil
}

// StopSimulation performs a clean shutdown of the simulated environment.
// This is intended to be called when the user exits the simulation view.
func (a *App) StopSimulation() {
	if a.simCancel != nil {
		a.simCancel()
		a.simCancel = nil
	}
 
	if a.activeComposePath != "" {
		stopDockerCompose(a.activeComposePath)
		a.activeComposePath = ""
	}
}

// shutdown is called by Wails before the window closes.
func (a *App) shutdown() {
	a.StopSimulation()
}

// stopDockerCompose runs `docker compose down --remove-orphans` to cleanly
// remove all containers started for the current simulation.
func stopDockerCompose(composePath string) {
	cmd := exec.Command("docker", "compose", "down", "--remove-orphans")
	cmd.Dir = dirOf(composePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Errors are logged but do not block the window from closing.
	if err := cmd.Run(); err != nil {
		fmt.Printf("[shutdown] docker compose down failed: %v\n", err)
	}
}

// launchDockerCompose runs `docker compose up -d` in the directory containing
// the given compose file.
func launchDockerCompose(composePath string) error {
	cmd := exec.Command("docker", "compose", "up", "--build", "-d")
	cmd.Dir = dirOf(composePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}
	return nil
}

// dirOf returns the directory component of a file path.
func dirOf(filePath string) string {
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			return filePath[:i]
		}
	}
	return "."
}
