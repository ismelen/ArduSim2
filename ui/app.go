package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	// activeStackName is set when running in Swarm mode so StopSimulation knows
	// which stack to remove via docker stack rm.
	activeStackName   string
	// activeSwarmHost is the DOCKER_HOST endpoint used for the running Swarm stack.
	activeSwarmHost   string
	// activeSimulationName stores the name of the current simulation for log organization.
	activeSimulationName string
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
	// Reset swarm state from any previous run.
	a.activeStackName = ""
	a.activeSwarmHost = ""

	log.Println(isLocal)
	
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
	} else {
		// SWARM MODE
		swarmHost := generalConfig.SwarmHost
		simName := generalConfig.SimulationName
		stackName := "Ardusim2-" + simName

		a.activeStackName = stackName
		a.activeSwarmHost = swarmHost
		a.activeSimulationName = simName

		if err := a.launchDockerStack(composePath, swarmHost, stackName); err != nil {
			return err
		}

		// Point the subscriber at the remote network_simulator.
		swarmIP, _, splitErr := strings.Cut(swarmHost, ":")
		if splitErr {
			a.subscriber.SetRemoteAddr(swarmIP)
		} else {
			// Fallback: use swarmHost as-is if no port separator found.
			a.subscriber.SetRemoteAddr(swarmHost)
		}

		// Prepare the subscriber fleet tracking.
		var uavIDs []string
		for _, uav := range uavs {
			uavIDs = append(uavIDs, uav.ID)
		}
		a.subscriber.SetExpectedFleet(uavIDs, nil)

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
	if config.OriginalSimulationName != "" && name != config.OriginalSimulationName {
		// Renamed. Do not delete the original, just start using the new name.
	}

	// Always just return targetPath so that previous timestamped runs accumulate.
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

// SaveSimulationConfig explicitly saves the configuration JSON state to a folder.
func (a *App) SaveSimulationConfig(uavs []simulation.UAV, generalConfig simulation.GeneralConfig, activeMode string) error {
	simDir, err := a.handleSimulationDirectory(&generalConfig)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(simDir, 0755); err != nil {
		return err
	}

	state := simulation.SimulationState{
		UAVs:          uavs,
		GeneralConfig: generalConfig,
		ActiveMode:    activeMode,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	statePath := filepath.Join(simDir, "simulation.json")
	return os.WriteFile(statePath, data, 0644)
}

// CleanSwarmNodes collects logs from all Swarm services and removes the stack.
func (a *App) CleanSwarmNodes() {
	if a.activeStackName == "" || a.activeSwarmHost == "" {
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	runDir := filepath.Join(a.paths.SimulationsDir, a.activeSimulationName, "runs", timestamp)

	dockerEnv := os.Environ()
	if !(strings.Contains(a.activeSwarmHost, "localhost") || strings.Contains(a.activeSwarmHost, "127.0.0.1")) {
		dockerEnv = append(dockerEnv, "DOCKER_HOST=tcp://"+a.activeSwarmHost)
	}

	// 1. Get all services in the stack
	cmd := exec.Command("docker", "stack", "services", "--format", "{{.Name}}", a.activeStackName)
	cmd.Env = dockerEnv
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[CleanSwarmNodes] failed to list services: %v\nOutput: %s\n", err, string(out))
		stopDockerStack(a.activeStackName, a.activeSwarmHost)
		return
	}

	serviceNames := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, fullName := range serviceNames {
		fullName = strings.TrimSpace(fullName)
		if fullName == "" {
			continue
		}

		nameWithoutPrefix := strings.TrimPrefix(fullName, a.activeStackName+"_")

		var destDir string
		if nameWithoutPrefix == "network_simulator" {
			destDir = filepath.Join(runDir, "network_simulator")
		} else {
			parts := strings.Split(nameWithoutPrefix, "_")
			if len(parts) >= 2 {
				uavID := parts[len(parts)-1]
				serviceType := strings.Join(parts[:len(parts)-1], "_")
				destDir = filepath.Join(runDir, "uav-"+uavID, serviceType)
			} else {
				destDir = filepath.Join(runDir, "unknown", nameWithoutPrefix)
			}
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("[CleanSwarmNodes] failed to create dir %s: %v\n", destDir, err)
			continue
		}

		logCmd := exec.Command("docker", "service", "logs", fullName)
		logCmd.Env = dockerEnv
		logOut, logErr := logCmd.CombinedOutput()
		if logErr != nil {
			fmt.Printf("[CleanSwarmNodes] failed to get logs for %s: %v\n", fullName, logErr)
			continue
		}

		logPath := filepath.Join(destDir, "service.log")
		if err := os.WriteFile(logPath, logOut, 0644); err != nil {
			fmt.Printf("[CleanSwarmNodes] failed to write log to %s: %v\n", logPath, err)
		}
	}

	stopDockerStack(a.activeStackName, a.activeSwarmHost)
}

// StopSimulation performs a clean shutdown of the simulated environment.
// This is intended to be called when the user exits the simulation view.
func (a *App) StopSimulation() {
	if a.simCancel != nil {
		a.simCancel()
		a.simCancel = nil
	}

	if a.activeStackName != "" {
		// SWARM MODE: collect logs and remove stack
		a.CleanSwarmNodes()
		a.activeStackName = ""
		a.activeSwarmHost = ""
		a.activeSimulationName = ""
	} else if a.activeComposePath != "" {
		stopDockerCompose(a.activeComposePath)
		a.activeComposePath = ""
	}
}

// DiscardCurrentRun deletes the runs/ folder in the simulation root and specific network logs to discard the current telemetry.
func (a *App) DiscardCurrentRun(generalConfig simulation.GeneralConfig) error {
	simDir, err := a.handleSimulationDirectory(&generalConfig)
	if err != nil {
		return err
	}

	// In orchestrator.go, runs are placed in "runs/*" under simDir.
	// Since we accumulate, if we want to discard *this* run specifically, we should probably know its exact timestamp.
	// However, if discarding means deleting the latest run... actually, we would need to know the run directory...
	// To simplify: if they want to discard the run, we can just delete the latest modified directory in simDir/runs.
	runsDir := filepath.Join(simDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err == nil && len(entries) > 0 {
		var latest os.DirEntry
		var latestTime time.Time
		for _, e := range entries {
			if e.IsDir() {
				info, err := e.Info()
				if err == nil && info.ModTime().After(latestTime) {
					latest = e
					latestTime = info.ModTime()
				}
			}
		}
		if latest != nil {
			os.RemoveAll(filepath.Join(runsDir, latest.Name()))
		}
	}
	return nil
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

// stopDockerStack removes a Swarm stack from the remote Docker host.
func stopDockerStack(stackName, swarmHost string) {
	cmd := exec.Command("docker", "stack", "rm", stackName)
	cmd.Env = append(os.Environ(), "DOCKER_HOST=tcp://"+swarmHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("[shutdown] docker stack rm failed: %v\n", err)
	}
}

// launchDockerStack deploys a Swarm stack on the remote Docker host and polls
// until all services are Running. Logs are streamed to the frontend via
// the "simulation:log" event.
//
// Images must already be built and available on the Swarm nodes; this function
// does not perform any build step.
func (a *App) launchDockerStack(swarmComposePath, swarmHost, stackName string) error {
	dockerEnv := os.Environ()
	if !(strings.Contains(swarmHost, "localhost") || strings.Contains(swarmHost, "127.0.0.1")) {
		dockerEnv = append(dockerEnv, "DOCKER_HOST=tcp://"+swarmHost)
	}
	
	// 1. Deploy the stack (non-detached — blocks until deploy command returns).
	deployCmd := exec.Command("docker", "stack", "deploy", "-c", swarmComposePath, stackName)
	deployCmd.Env = dockerEnv
	deployStdout, _ := deployCmd.StdoutPipe()
	deployStderr, _ := deployCmd.StderrPipe()

	if err := deployCmd.Start(); err != nil {
		return fmt.Errorf("docker stack deploy start failed: %w", err)
	}

	// Stream deploy logs synchronously so the user sees progress.
	deployScanner := simulation.NewLogScanner(deployStdout, deployStderr)
	for deployScanner.Scan() {
		runtime.EventsEmit(a.ctx, "simulation:log", deployScanner.Text())
	}

	if err := deployCmd.Wait(); err != nil {
		log.Println(err, deployCmd.Args)
		return fmt.Errorf("docker stack deploy failed: %w", err)
	}

	// 2. Poll until all services in the stack are Running.
	// go a.pollSwarmStackReady(dockerEnv, stackName)

	return nil
}

// pollSwarmStackReady polls `docker stack ps` until all tasks in the given
// stack are in state "Running", emitting each poll result as a simulation:log.
func (a *App) pollSwarmStackReady(dockerEnv []string, stackName string) {
	const (
		maxAttempts = 60
		pollDelay   = 5 * time.Second
	)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(pollDelay)

		cmd := exec.Command("docker", "stack", "ps", "--no-trunc", stackName)
		cmd.Env = dockerEnv
		out, err := cmd.CombinedOutput()
		if err != nil {
			runtime.EventsEmit(a.ctx, "simulation:log",
				fmt.Sprintf("[swarm] stack ps error: %v", err))
			continue
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			runtime.EventsEmit(a.ctx, "simulation:log", "[swarm] "+line)
		}

		// Check if every task (non-header line) contains "Running".
		allRunning := true
		taskLines := 0
		for _, line := range lines[1:] { // skip header
			if strings.TrimSpace(line) == "" {
				continue
			}
			taskLines++
			if !strings.Contains(line, "Running") {
				allRunning = false
			}
		}

		if taskLines > 0 && allRunning {
			runtime.EventsEmit(a.ctx, "simulation:log",
				fmt.Sprintf("[swarm] ✓ All services in stack %q are Running", stackName))
			return
		}
	}

	runtime.EventsEmit(a.ctx, "simulation:log",
		fmt.Sprintf("[swarm] ⚠ Timeout waiting for all services in %q to reach Running state", stackName))
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
