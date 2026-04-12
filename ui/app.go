package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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
	activeComposePath  string
	// activeAlgorithmIDs is the set of algorithm service IDs running in the current session.
	activeAlgorithmIDs map[string]bool
	// stoppedAlgorithmIDs tracks which algorithms the user has manually stopped.
	stoppedAlgorithmIDs map[string]bool
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
	a.subscriber.SetContext(ctx)
}

// GetAvailableServices returns all algorithm services that expose a schema.json.
func (a *App) GetAvailableServices() []simulation.ServiceType {
	return a.discoverer.GetAvailableServices()
}

// StartSimulation generates the Docker Compose environment for the given fleet
// and, when isLocal is true, launches it and subscribes to network telemetry.
func (a *App) StartSimulation(uavs []simulation.UAV, generalConfig simulation.GeneralConfig, activeMode string, isLocal bool) error {

	simDir, err := a.handleSimulationDirectory(&generalConfig)
	if err != nil {
		return fmt.Errorf("handle simulation directory: %w", err)
	}

	orchestrator := simulation.NewOrchestrator(a.paths)

	composePath, err := orchestrator.Run(uavs, generalConfig, activeMode, isLocal, simDir)
	if err != nil {
		return fmt.Errorf("prepare simulation: %w", err)
	}

	a.activeComposePath = composePath

	if isLocal {
		if err := a.launchDockerCompose(composePath); err != nil {
			return err
		}

		// Prepare the subscriber to track the fleet
		var uavIDs []string
		for _, uav := range uavs {
			uavIDs = append(uavIDs, uav.ID)
		}
		a.subscriber.SetExpectedFleet(uavIDs, nil)

		// Track the active algorithm IDs for manual stop detection.
		algorithmIDs := collectAlgorithmIDs(uavs)
		a.activeAlgorithmIDs = make(map[string]bool, len(algorithmIDs))
		a.stoppedAlgorithmIDs = make(map[string]bool)
		for _, id := range algorithmIDs {
			a.activeAlgorithmIDs[id] = true
		}

		// When all UAVs emit "finish", also broadcast stop to all algorithms.
		a.subscriber.SetOnFinish(func() {
			for _, algo := range algorithmIDs {
				if err := a.SendAlgorithmCommand(algo, "stop"); err != nil {
					fmt.Printf("[app] failed to stop algorithm %s: %v\n", algo, err)
				}
			}
		})

		// Create a session-specific context that can be cancelled without killing the app.
		if a.simCancel != nil {
			a.simCancel()
		}
		a.simCtx, a.simCancel = context.WithCancel(a.ctx)
		go a.subscriber.Start(a.simCtx)
	}

	return nil
}

func (a *App) handleSimulationDirectory(config *simulation.GeneralConfig) (string, error) {
	name := strings.TrimSpace(config.SimulationName)
	if name == "" {
		name = time.Now().Format("20060102_150405")
	} else {
		// Just remove anything outside azAZ0-9 _ -
		reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]+`)
		name = reg.ReplaceAllString(name, "_")
	}
	config.SimulationName = name

	targetPath := filepath.Join(a.paths.SimulationsDir, name)

	// User loaded this config from a folder
	if config.OriginalSimulationName != "" {
		originalPath := filepath.Join(a.paths.SimulationsDir, config.OriginalSimulationName)
		if name != config.OriginalSimulationName {
			// Renamed. Delete the old one.
			os.RemoveAll(originalPath)
		}
		// Reset resources for this one
		os.RemoveAll(targetPath)
		return targetPath, nil
	}

	// New config
	if _, err := os.Stat(targetPath); err == nil {
		res, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Carpeta Existente",
			Message:       fmt.Sprintf("La simulación '%s' ya existe. ¿Quieres reemplazarla? Si pulsas No, se creará una copia.", name),
			DefaultButton: "Yes",
		})
		if err != nil {
			return "", err
		}
		if res == "Yes" {
			os.RemoveAll(targetPath)
		} else {
			for i := 2; ; i++ {
				newName := fmt.Sprintf("%s(%d)", name, i)
				newPath := filepath.Join(a.paths.SimulationsDir, newName)
				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					name = newName
					targetPath = newPath
					config.SimulationName = name
					break
				}
			}
		}
	}
	return targetPath, nil
}

// collectAlgorithmIDs returns the unique algorithm service IDs across all UAVs.
func collectAlgorithmIDs(uavs []simulation.UAV) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, uav := range uavs {
		for _, svc := range uav.Services {
			if !seen[svc.ServiceId] {
				seen[svc.ServiceId] = true
				ids = append(ids, svc.ServiceId)
			}
		}
	}
	return ids
}

// SendAlgorithmCommand broadcasts a command message to the specified algorithm of all UAVs.
// When the user manually stops all active algorithms, it notifies the subscriber
// to trigger the simulation-finished flow.
func (a *App) SendAlgorithmCommand(serviceId string, command string) error {
	payload := map[string]interface{}{
		"topic":   "algo/" + serviceId,
		"payload": map[string]interface{}{
			"command": command,
		},
	}
	if err := a.subscriber.SendGlobalBroadcast(payload); err != nil {
		return err
	}

	// Track manual stops to detect when the user has stopped all algorithms.
	if command == "stop" && len(a.activeAlgorithmIDs) > 0 {
		a.stoppedAlgorithmIDs[serviceId] = true
		allStopped := true
		for id := range a.activeAlgorithmIDs {
			if !a.stoppedAlgorithmIDs[id] {
				allStopped = false
				break
			}
		}
		if allStopped {
			a.subscriber.NotifyUserStoppedAll(a.ctx)
		}
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

	// Repair missing FolderName if loading an old simulation config
	availableServices := a.GetAvailableServices()
	svcMap := make(map[string]string)
	for _, s := range availableServices {
		svcMap[s.ID] = s.FolderName
	}

	for i := range state.UAVs {
		for j := range state.UAVs[i].Services {
			if state.UAVs[i].Services[j].FolderName == "" {
				if folder, ok := svcMap[state.UAVs[i].Services[j].ServiceId]; ok {
					state.UAVs[i].Services[j].FolderName = folder
				} else {
					// Fallback to ServiceId if not found in current discovered services
					state.UAVs[i].Services[j].FolderName = state.UAVs[i].Services[j].ServiceId
				}
			}
		}
	}

	state.GeneralConfig.OriginalSimulationName = filepath.Base(selectedDir)
	state.GeneralConfig.SimulationName = filepath.Base(selectedDir)

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

// SelectFile opens a native file picker and returns the absolute path
// to the selected file. This is required because browser-based file inputs
// only provide the filename for security reasons.
func (a *App) SelectFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Auxiliary File",
		Filters: []runtime.FileFilter{
			{DisplayName: "KML files (*.kml)", Pattern: "*.kml"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
}

// GetKmlFirstCoordinate extracts the first coordinate (Lat, Lon, Alt) from a KML file.
func (a *App) GetKmlFirstCoordinate(path string) (*simulation.Coordinate, error) {
	return simulation.GetFirstCoordinate(path)
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
// the given compose file and streams logs to the frontend.
func (a *App) launchDockerCompose(composePath string) error {
	// 1. Quick check if Docker is running
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("DOCKER_NOT_RUNNING: please start Docker Desktop or the Docker daemon")
	}

	cmd := exec.Command("docker", "compose", "up", "--build", "-d")
	cmd.Dir = dirOf(composePath)

	// Capture stdout and stderr to stream to UI
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("docker compose start failed: %w", err)
	}

	// Stream logs in background
	go func() {
		scanner := simulation.NewLogScanner(stdout, stderr)
		for scanner.Scan() {
			runtime.EventsEmit(a.ctx, "simulation:log", scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
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
