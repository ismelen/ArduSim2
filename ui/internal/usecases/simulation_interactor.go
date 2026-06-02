package usecases

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ui/internal/domain"
	"ui/internal/infrastructure/kml"
	"ui/internal/infrastructure/util"
	"ui/internal/ports"
)

type SimulationInteractor struct {
	orchestrator ports.ContainerOrchestrator
	subscriber   ports.TelemetrySubscriber
	repo         ports.ConfigRepository
	ui           ports.UIBridge

	// State for the active simulation session
	activeComposePath    string
	activeStackName       string
	activeSwarmHost       string
	activeSimulationName  string
	activeUAVIDs          []string
	activeAlgorithmIDs    map[string]bool
	stoppedAlgorithmIDs   map[string]bool
}

func NewSimulationInteractor(
	orchestrator ports.ContainerOrchestrator,
	subscriber ports.TelemetrySubscriber,
	repo ports.ConfigRepository,
	ui ports.UIBridge,
) *SimulationInteractor {
	return &SimulationInteractor{
		orchestrator:        orchestrator,
		subscriber:          subscriber,
		repo:                repo,
		ui:                  ui,
		activeAlgorithmIDs:  make(map[string]bool),
		stoppedAlgorithmIDs: make(map[string]bool),
	}
}

func (i *SimulationInteractor) StartSimulation(ctx context.Context, uavs []domain.UAV, config domain.GeneralConfig, isLocal bool) error {
	config.SanitizeSimulationName()
	simDir := filepath.Join(i.repo.GetSimulationsDir(), config.SimulationName)

	composePath, err := i.orchestrator.Run(uavs, config, isLocal, simDir)
	if err != nil {
		return fmt.Errorf("prepare simulation: %w", err)
	}

	i.activeComposePath = composePath
	i.activeStackName = ""
	i.activeSwarmHost = ""
	i.activeSimulationName = config.SimulationName
	i.activeUAVIDs = make([]string, len(uavs))
	for idx, uav := range uavs {
		i.activeUAVIDs[idx] = uav.ID
	}

	if isLocal {
		if err := i.orchestrator.StartCompose(composePath); err != nil {
			return err
		}

		uavIDs := make([]string, len(uavs))
		for idx, uav := range uavs {
			uavIDs[idx] = uav.ID
		}
		i.subscriber.SetExpectedFleet(uavIDs)

		// Algorithm tracking
		algoIDs := util.CollectAlgorithmIDs(uavs)
		i.activeAlgorithmIDs = make(map[string]bool)
		i.stoppedAlgorithmIDs = make(map[string]bool)
		for _, id := range algoIDs {
			i.activeAlgorithmIDs[id] = true
		}

		i.subscriber.SetOnFinish(func() {
			for _, algo := range algoIDs {
				if err := i.SendAlgorithmCommand(algo, "stop"); err != nil {
					fmt.Printf("[interactor] failed to stop algorithm %s: %v\n", algo, err)
				}
			}
		})
	} else {
		// SWARM MODE
		stackName := "Ardusim2-" + config.SimulationName
		i.activeStackName = stackName
		i.activeSwarmHost = config.SwarmHost

		if err := i.orchestrator.StartStack(composePath, config.SwarmHost, stackName); err != nil {
			return err
		}

		swarmIP, _, _ := strings.Cut(config.SwarmHost, ":")
		i.subscriber.SetRemoteAddr(swarmIP)

		uavIDs := make([]string, len(uavs))
		for idx, uav := range uavs {
			uavIDs[idx] = uav.ID
		}
		i.subscriber.SetExpectedFleet(uavIDs)
	}
	go i.subscriber.Start(ctx)

	return nil
}

func (i *SimulationInteractor) StopSimulation() {
	if i.activeStackName != "" {
		_ = i.orchestrator.CollectSwarmLogs(i.activeStackName, i.activeSwarmHost, i.activeSimulationName, "")
		_ = i.orchestrator.StopStack(i.activeStackName, i.activeSwarmHost)
		
		i.activeStackName = ""
		i.activeSwarmHost = ""
		i.activeSimulationName = ""
	} else if i.activeComposePath != "" {
		_ = i.orchestrator.StopCompose(i.activeComposePath)
		i.activeComposePath = ""
	}
}

func (i *SimulationInteractor) SendAlgorithmCommand(serviceId string, command string) error {
	payload := map[string]interface{}{
		"topic":   "algo/" + serviceId,
		"payload": map[string]interface{}{
			"command": command,
		},
	}
	if err := i.subscriber.SendGlobalBroadcast(payload); err != nil {
		return err
	}

	i.ui.EmitEvent("netsim:message", map[string]string{
		"source":  "[Global] ",
		"label":   fmt.Sprintf("[Global] Command '%s' sent to service: %s", command, serviceId),
	})

	if command == "stop" && len(i.activeAlgorithmIDs) > 0 {
		i.stoppedAlgorithmIDs[serviceId] = true
		allStopped := true
		for id := range i.activeAlgorithmIDs {
			if !i.stoppedAlgorithmIDs[id] {
				allStopped = false
				break
			}
		}
		if allStopped {
			// Notify subscriber via context? Actually app.go used a.ctx
			// We should probably pass a context or use one from start
			i.subscriber.NotifyUserStoppedAll(context.Background()) // Simplified for now
		}
	}
	return nil
}

func (i *SimulationInteractor) DownloadLogs() error {
	if i.activeSimulationName == "" {
		return fmt.Errorf("no active simulation")
	}
	simDir := filepath.Join(i.repo.GetSimulationsDir(), i.activeSimulationName)
	logsDir := filepath.Join(simDir, "logs")
	os.MkdirAll(logsDir, 0755)

	// PASO 1: Descargar ZIP del logger a memoria
	loggerZipBytes, err := i.fetchLoggerZip()
	if err != nil {
		return fmt.Errorf("fetch logger zip: %w", err)
	}

	// PASO 2: Recoger rutas de logs de ArduPilot (ya en host via bind-mount)
	uavLogDirs := map[string]string{}
	for _, uavID := range i.activeUAVIDs {
		dir := filepath.Join(simDir, "uav_logs", uavID)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			uavLogDirs[uavID] = dir
		}
	}

	// PASO 3 + 4: Construir y guardar ZIP combinado
	timestamp := time.Now().Format("20060102_150405")
	outPath := filepath.Join(logsDir, fmt.Sprintf("logs_%s.zip", timestamp))
	return i.buildCombinedZip(loggerZipBytes, uavLogDirs, outPath)
}

func (i *SimulationInteractor) fetchLoggerZip() ([]byte, error) {
	url := "http://localhost:8080/api/logs/download"
	if i.activeSwarmHost != "" && !strings.Contains(i.activeSwarmHost, "localhost") && !strings.Contains(i.activeSwarmHost, "127.0.0.1") {
		host, _, _ := strings.Cut(i.activeSwarmHost, ":")
		url = fmt.Sprintf("http://%s:8080/api/logs/download", host)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to reach logger service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("logger service returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (i *SimulationInteractor) buildCombinedZip(loggerZipBytes []byte, uavLogDirs map[string]string, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create combined zip: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// 1. Copy original logger zip contents
	if len(loggerZipBytes) > 0 {
		loggerReader, err := zip.NewReader(bytes.NewReader(loggerZipBytes), int64(len(loggerZipBytes)))
		if err != nil {
			return fmt.Errorf("read logger zip: %w", err)
		}

		for _, file := range loggerReader.File {
			rc, err := file.Open()
			if err != nil {
				return err
			}
			header := file.FileHeader
			writer, err := zipWriter.CreateHeader(&header)
			if err != nil {
				rc.Close()
				return err
			}
			if _, err := io.Copy(writer, rc); err != nil {
				rc.Close()
				return err
			}
			rc.Close()
		}
	}

	// 2. Add ArduPilot logs
	for uavID, dir := range uavLogDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return nil
			}

			// Clean path for ZIP
			relPath = filepath.ToSlash(relPath)
			zipPath := fmt.Sprintf("ardupilot/%s/%s", uavID, relPath)

			fileInfo, err := d.Info()
			if err != nil {
				return nil
			}

			header, err := zip.FileInfoHeader(fileInfo)
			if err != nil {
				return nil
			}
			header.Name = zipPath
			header.Method = zip.Deflate

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			return err
		})
		if err != nil {
			fmt.Printf("[interactor] error walking uav_logs for %s: %v\n", uavID, err)
		}
	}

	return nil
}

func (i *SimulationInteractor) SearchLogs(zipPath string, filter domain.LogFilter) ([]domain.LogMessage, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %v", err)
	}
	defer r.Close()

	var allLogs []domain.LogMessage
	filterText := strings.ToLower(filter.SearchText)

	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".jsonl") {
			continue
		}
		
		rc, err := f.Open()
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			var msg domain.LogMessage
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}

			// Apply filters
			if filter.InstanceID != "" && !strings.Contains(strings.ToLower(msg.InstanceID), strings.ToLower(filter.InstanceID)) {
				continue
			}
			if filter.ServiceID != "" && !strings.Contains(strings.ToLower(msg.ServiceID), strings.ToLower(filter.ServiceID)) {
				continue
			}
			if filter.Level != "" && !strings.EqualFold(msg.Level, filter.Level) {
				continue
			}
			if filter.EventID != "" && !strings.Contains(strings.ToLower(msg.EventID), strings.ToLower(filter.EventID)) {
				continue
			}
			if filterText != "" && !strings.Contains(strings.ToLower(msg.Message), filterText) {
				continue
			}

			allLogs = append(allLogs, msg)
		}
		rc.Close()
	}

	// Sort logs chronologically (oldest first, so newest at the bottom as requested)
	sort.Slice(allLogs, func(a, b int) bool {
		return allLogs[a].Timestamp.Before(allLogs[b].Timestamp)
	})

	return allLogs, nil
}

func (i *SimulationInteractor) LoadLogEntries(ctx context.Context) ([]string, error) {
	baseDir := i.repo.GetSimulationsDir()
	var entries []string

	entriesMap := make(map[string]bool)

	// We expect logs to be in simulations/<sim_name>/logs/logs_*.zip
	entriesFiles, err := filepath.Glob(filepath.Join(baseDir, "*", "logs", "*.zip"))
	if err == nil {
		for _, file := range entriesFiles {
			entriesMap[file] = true
		}
	}

	for k := range entriesMap {
		entries = append(entries, k)
	}

	// Sort entries reverse chronologically so newest zip is at the top
	sort.Sort(sort.Reverse(sort.StringSlice(entries)))

	return entries, nil
}

func (i *SimulationInteractor) LoadFile(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	return string(bytes), err
}

func (i *SimulationInteractor) LoadSimulationConfig(ctx context.Context) (*domain.SimulationState, error) {
	selectedDir, err := i.ui.OpenDirectoryDialog(ctx, "Select Simulation Directory", i.repo.GetSimulationsDir())
	if err != nil || selectedDir == "" {
		return nil, err
	}

	stateFile := filepath.Join(selectedDir, "simulation.json")
	state, err := i.repo.LoadSimulation(stateFile)
	if err != nil {
		return nil, err
	}

	// Repair logic
	availableServices := i.repo.GetAvailableServices()
	svcMap := make(map[string]string)
	for _, s := range availableServices {
		svcMap[s.ID] = s.FolderName
	}

	for idxUAV := range state.UAVs {
		for idxSvc := range state.UAVs[idxUAV].Services {
			svc := &state.UAVs[idxUAV].Services[idxSvc]
			if svc.FolderName == "" {
				if folder, ok := svcMap[svc.ServiceId]; ok {
					svc.FolderName = folder
				} else {
					svc.FolderName = svc.ServiceId
				}
			}
		}
	}

	state.GeneralConfig.OriginalSimulationName = filepath.Base(selectedDir)
	state.GeneralConfig.SimulationName = filepath.Base(selectedDir)

	return state, nil
}

func (i *SimulationInteractor) SaveSimulationConfig(uavs []domain.UAV, config domain.GeneralConfig, mode string) error {
	config.SanitizeSimulationName()
	simDir := filepath.Join(i.repo.GetSimulationsDir(), config.SimulationName)

	state := domain.SimulationState{
		UAVs:          uavs,
		GeneralConfig: config,
		ActiveMode:    mode,
	}

	return i.repo.SaveSimulation(simDir, state)
}

func (i *SimulationInteractor) DiscardCurrentRun(config domain.GeneralConfig) error {
	config.SanitizeSimulationName()
	// This logic remains similar, but using the repo
	return nil // Implementation detail for later
}

func (i *SimulationInteractor) SelectFile(ctx context.Context) (string, error) {
	return i.ui.OpenFileDialog(ctx, "Select Auxiliary File", []ports.FileFilter{
		{DisplayName: "KML files (*.kml)", Pattern: "*.kml"},
		{DisplayName: "All files", Pattern: "*.*"},
	})
}

func (i *SimulationInteractor) SelectSpeedProfile(ctx context.Context) (string, error) {
	return i.ui.OpenFileDialog(ctx, "Select Speed Profile", []ports.FileFilter{
		{DisplayName: "Speed profile files (*.dat)", Pattern: "*.dat"},
		{DisplayName: "All files", Pattern: "*.*"},
	})
}

func (i *SimulationInteractor) GetKmlFirstCoordinate(path string) (*domain.Coordinate, error) {
	return kml.GetFirstCoordinate(path)
}
