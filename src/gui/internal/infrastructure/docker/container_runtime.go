package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ui/internal/infrastructure/util"
	"ui/internal/ports"
)

type DockerRuntime struct {
	ui             ports.UIBridge
	algorithmsDir  string
	mixersDir      string
	controllersDir string
}

func NewDockerRuntime(projectRoot string, ui ports.UIBridge) ports.ContainerRuntime {
	base := filepath.Clean(projectRoot)
	return &DockerRuntime{
		ui:             ui,
		algorithmsDir:  filepath.Join(base, "..", "algorithms"),
		mixersDir:      filepath.Join(base, "..", "mixers"),
		controllersDir: filepath.Join(base, "..", "controllers"),
	}
}

func (r *DockerRuntime) StartCompose(composePath string) error {
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("DOCKER_NOT_RUNNING")
	}

	cmd := exec.Command("docker", "compose", "-f", filepath.Base(composePath), "up", "--build", "-d")
	cmd.Dir = filepath.Dir(composePath)

	var stderrBuf bytes.Buffer
	stdout, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := util.NewLogScanner(stdout, io.TeeReader(stderrPipe, &stderrBuf))
		for scanner.Scan() {
			r.ui.EmitEvent("simulation:log", scanner.Text())
		}
	}()

	waitErr := cmd.Wait()
	<-done

	if waitErr != nil {
		details := strings.TrimSpace(stderrBuf.String())
		if details != "" {
			return fmt.Errorf("docker compose: %w\n%s", waitErr, details)
		}
		return fmt.Errorf("docker compose: %w", waitErr)
	}
	return nil
}

func (r *DockerRuntime) StopCompose(composePath string) error {
	cmd := exec.Command("docker", "compose", "-f", filepath.Base(composePath), "down", "--remove-orphans", "--volumes")
	cmd.Dir = filepath.Dir(composePath)
	return cmd.Run()
}

func (r *DockerRuntime) buildComposeServices(composePath string, services ...string) error {
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("DOCKER_NOT_RUNNING")
	}

	args := []string{"compose", "-f", filepath.Base(composePath), "build"}
	args = append(args, services...)
	cmd := exec.Command("docker", args...)
	cmd.Dir = filepath.Dir(composePath)

	var stderrBuf bytes.Buffer
	stdout, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := util.NewLogScanner(stdout, io.TeeReader(stderrPipe, &stderrBuf))
		for scanner.Scan() {
			r.ui.EmitEvent("simulation:log", scanner.Text())
		}
	}()

	waitErr := cmd.Wait()
	<-done

	if waitErr != nil {
		details := strings.TrimSpace(stderrBuf.String())
		if details != "" {
			return fmt.Errorf("docker compose build: %w\n%s", waitErr, details)
		}
		return fmt.Errorf("docker compose build: %w", waitErr)
	}
	if len(services) == 0 {
		r.ui.EmitEvent("simulation:log", "[Build] Images successfully built.")
	} else {
		r.ui.EmitEvent("simulation:log", fmt.Sprintf("[Build] Service(s) %s successfully built.", strings.Join(services, ", ")))
	}
	return nil
}

func (r *DockerRuntime) BuildCompose(composePath string) error {
	return r.buildComposeServices(composePath)
}

func (r *DockerRuntime) GenerateBuildManifest(simDir string, dockerHubRepository string, isKubernetes bool, ardupilotPath string) error {
	builder := NewComposeBuilder()
	
	prefixImage := func(img string) string {
		if isKubernetes && dockerHubRepository != "" {
			return dockerHubRepository + ":" + img
		}
		return img
	}

	absPath := func(relPath string) string {
		abs, err := filepath.Abs(relPath)
		if err == nil {
			return strings.ReplaceAll(abs, "\\", "/")
		}
		return relPath
	}

	// Core images
	builder.AddService(ports.ComposeService{
		Name:  "netsim_gateway",
		Image: prefixImage("netsim_gateway"),
		Build: &ports.ComposeBuild{Context: absPath("../../src/netsim_gateway"), Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "netsim",
		Image: prefixImage("netsim"),
		Build: &ports.ComposeBuild{Context: absPath("../../src/netsim"), Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "logger",
		Image: prefixImage("logger"),
		Build: &ports.ComposeBuild{Context: absPath("../../src/logger"), Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "communication_module",
		Image: prefixImage("communication_module"),
		Build: &ports.ComposeBuild{Context: absPath("../../src/communication_module"), Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "external_comms",
		Image: prefixImage("external_comms"),
		Build: &ports.ComposeBuild{Context: absPath("../../src/external_comms"), Dockerfile: "Dockerfile"},
	})

	// Build base uav_controller image
	builder.AddService(ports.ComposeService{
		Name:  "uav_controller_base",
		Image: prefixImage("uav_controller_base:latest"),
		Build: &ports.ComposeBuild{
			Context:    absPath("../../src/uav_controller/ardupilot4_5_3"),
			Dockerfile: "SITL",
		},
	})

	binName := filepath.Base(ardupilotPath)
	uavControllerImage := "uav_controller"
	if binName != "" {
		uavControllerImage = fmt.Sprintf("uav_controller_%s", strings.ToLower(strings.ReplaceAll(binName, ".", "_")))
	}

	resDir := filepath.Join(simDir, "resources")
	os.MkdirAll(resDir, 0755)

	dockerfileContent := fmt.Sprintf("FROM %s\nCOPY %s /app/arducopter\nRUN chmod +x /app/arducopter\n", prefixImage("uav_controller_base:latest"), binName)
	_ = os.WriteFile(filepath.Join(resDir, "Dockerfile.uav_controller"), []byte(dockerfileContent), 0644)

	builder.AddService(ports.ComposeService{
		Name:      uavControllerImage,
		Image:     prefixImage(uavControllerImage),
		DependsOn: []string{"uav_controller_base"},
		Build: &ports.ComposeBuild{
			Context:    "./resources",
			Dockerfile: "Dockerfile.uav_controller",
		},
	})

	appendServices := func(dir, category string, dockerfile string) {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					folderName := entry.Name()
					imageName := folderName

					schemaPath := filepath.Join(dir, folderName, "schema.json")
					if rawData, err := os.ReadFile(schemaPath); err == nil {
						var schema struct {
							ServiceID string `json:"service_id"`
						}
						if err := json.Unmarshal(rawData, &schema); err == nil && schema.ServiceID != "" {
							imageName = schema.ServiceID
						}
					}

					builder.AddService(ports.ComposeService{
						Name:  imageName,
						Image: prefixImage(imageName),
						Build: &ports.ComposeBuild{Context: absPath(fmt.Sprintf("../../src/%s/%s", category, folderName)), Dockerfile: dockerfile},
					})
				}
			}
		}
	}

	appendServices(r.algorithmsDir, "algorithms", "Dockerfile")
	appendServices(r.mixersDir, "mixers", "Dockerfile")

	composePath := filepath.Join(simDir, "docker-compose.build.yaml")
	if err := os.WriteFile(composePath, []byte(builder.Build()), 0644); err != nil {
		return fmt.Errorf("failed to write build compose file: %w", err)
	}

	return nil
}

func (r *DockerRuntime) BuildAllImages(simDir string, dockerHubRepository string, isKubernetes bool, ardupilotPath string) error {
	if err := r.GenerateBuildManifest(simDir, dockerHubRepository, isKubernetes, ardupilotPath); err != nil {
		return err
	}

	composePath := filepath.Join(simDir, "docker-compose.build.yaml")

	r.ui.EmitEvent("simulation:log", "[Build] Pre-building uav_controller_base...")
	if err := r.buildComposeServices(composePath, "uav_controller_base"); err != nil {
		return fmt.Errorf("failed to pre-build uav_controller_base: %w", err)
	}

	if err := r.BuildCompose(composePath); err != nil {
		return err
	}

	if isKubernetes {
		r.ui.EmitEvent("simulation:log", "[Push] Publishing images to registry...")
		pushCmd := exec.Command("docker", "compose", "-f", filepath.Base(composePath), "push")
		pushCmd.Dir = filepath.Dir(composePath)
		var pushStderrBuf bytes.Buffer
		pushStdout, _ := pushCmd.StdoutPipe()
		pushStderrPipe, _ := pushCmd.StderrPipe()
		
		if err := pushCmd.Start(); err != nil {
			return err
		}

		donePush := make(chan struct{})
		go func() {
			defer close(donePush)
			scanner := util.NewLogScanner(pushStdout, io.TeeReader(pushStderrPipe, &pushStderrBuf))
			for scanner.Scan() {
				r.ui.EmitEvent("simulation:log", scanner.Text())
			}
		}()

		waitErr := pushCmd.Wait()
		<-donePush

		if waitErr != nil {
			details := strings.TrimSpace(pushStderrBuf.String())
			if details != "" {
				return fmt.Errorf("docker compose push: %w\n%s", waitErr, details)
			}
			return fmt.Errorf("docker compose push: %w", waitErr)
		}
		r.ui.EmitEvent("simulation:log", "[Push] Images successfully published.")
	}

	return nil
}

func (r *DockerRuntime) runStreamedCommand(cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args...)
	var stderrBuf bytes.Buffer
	stdout, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()
	
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := util.NewLogScanner(stdout, io.TeeReader(stderrPipe, &stderrBuf))
		for scanner.Scan() {
			r.ui.EmitEvent("simulation:log", scanner.Text())
		}
	}()

	waitErr := cmd.Wait()
	<-done

	if waitErr != nil {
		details := strings.TrimSpace(stderrBuf.String())
		if details != "" {
			return fmt.Errorf("%s: %w\n%s", cmdName, waitErr, details)
		}
		return fmt.Errorf("%s: %w", cmdName, waitErr)
	}
	return nil
}

func (r *DockerRuntime) StartKubernetes(manifestPath, dockerHubRepository, kubeConfigPath string) (string, string, error) {
	r.ui.EmitEvent("simulation:log", "[Kubernetes] Applying manifest: "+manifestPath)

	args := []string{"apply", "-f", manifestPath}
	if kubeConfigPath != "" {
		args = append(args, "--kubeconfig="+kubeConfigPath)
	}

	if err := r.runStreamedCommand("kubectl", args...); err != nil {
		return "", "", err
	}

	r.ui.EmitEvent("simulation:log", "[Kubernetes] Waiting for logger pod to be Ready (120s timeout)...")
	waitLoggerArgs := []string{"wait", "--for=condition=Ready", "pod", "-l", "app=logger", "--timeout=120s"}
	if kubeConfigPath != "" {
		waitLoggerArgs = append(waitLoggerArgs, "--kubeconfig="+kubeConfigPath)
	}
	if err := r.runStreamedCommand("kubectl", waitLoggerArgs...); err != nil {
		r.ui.EmitEvent("simulation:log", "[Kubernetes] Warning: logger pod wait failed: "+err.Error())
	}

	r.ui.EmitEvent("simulation:log", "[Kubernetes] Waiting for netsim-gateway pod to be Ready (120s timeout)...")
	waitNetsimArgs := []string{"wait", "--for=condition=Ready", "pod", "-l", "app=netsim-gateway", "--timeout=120s"}
	if kubeConfigPath != "" {
		waitNetsimArgs = append(waitNetsimArgs, "--kubeconfig="+kubeConfigPath)
	}
	if err := r.runStreamedCommand("kubectl", waitNetsimArgs...); err != nil {
		r.ui.EmitEvent("simulation:log", "[Kubernetes] Warning: netsim-gateway pod wait failed: "+err.Error())
	}

	getIP := func(appLabel string) string {
		getArgs := []string{"get", "pods", "-l", "app="+appLabel, "-o", "jsonpath={.items[0].status.podIP}"}
		if kubeConfigPath != "" {
			getArgs = append(getArgs, "--kubeconfig="+kubeConfigPath)
		}
		out, _ := exec.Command("kubectl", getArgs...).Output()
		return strings.Trim(string(out), " '\n\r\"")
	}

	loggerIP := getIP("logger")
	gatewayIP := getIP("netsim-gateway")

	return loggerIP, gatewayIP, nil
}

func (r *DockerRuntime) StopKubernetes(manifestPath, kubeConfigPath string) error {
	r.ui.EmitEvent("simulation:log", "[Kubernetes] Deleting resources from manifest: "+manifestPath)
	
	args := []string{"delete", "-f", manifestPath}
	if kubeConfigPath != "" {
		args = append(args, "--kubeconfig="+kubeConfigPath)
	}

	return r.runStreamedCommand("kubectl", args...)
}

func (r *DockerRuntime) CollectKubernetesLogs(simName, destDir string) error {
	r.ui.EmitEvent("simulation:log", "Kubernetes log collection not yet implemented.")
	return nil
}
