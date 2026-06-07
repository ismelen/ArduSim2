package docker

import (
	"bytes"
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

func (r *DockerRuntime) BuildCompose(composePath string) error {
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("DOCKER_NOT_RUNNING")
	}

	cmd := exec.Command("docker", "compose", "-f", filepath.Base(composePath), "build")
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
	r.ui.EmitEvent("simulation:log", "[Build] Images successfully built.")
	return nil
}

func (r *DockerRuntime) BuildAllImages(simDir string, dockerHubRepository string, isKubernetes bool, ardupilotPath string) error {
	builder := NewComposeBuilder()
	
	prefixImage := func(img string) string {
		if isKubernetes && dockerHubRepository != "" {
			return dockerHubRepository + ":" + img
		}
		return img
	}

	// Core images
	builder.AddService(ports.ComposeService{
		Name:  "netsim_gateway",
		Image: prefixImage("netsim_gateway"),
		Build: &ports.ComposeBuild{Context: "../../src/netsim_gateway", Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "netsim",
		Image: prefixImage("netsim"),
		Build: &ports.ComposeBuild{Context: "../../src/netsim", Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "logger",
		Image: prefixImage("logger"),
		Build: &ports.ComposeBuild{Context: "../../src/logger", Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "communication_module",
		Image: prefixImage("communication_module"),
		Build: &ports.ComposeBuild{Context: "../../src/communication_module", Dockerfile: "Dockerfile"},
	})
	builder.AddService(ports.ComposeService{
		Name:  "external_comms",
		Image: prefixImage("external_comms"),
		Build: &ports.ComposeBuild{Context: "../../src/external_comms", Dockerfile: "Dockerfile"},
	})
	// Attempt to make ardupilotPath relative to the context directory
	contextDir, _ := filepath.Abs("../../src/uav_controller/ardupilot4_5_3")
	relPath, err := filepath.Rel(contextDir, ardupilotPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		relPath = ardupilotPath
	}
	// On Windows, Docker paths inside compose args should use forward slashes
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	builder.AddService(ports.ComposeService{
		Name:  "uav_controller",
		Image: prefixImage("uav_controller"),
		Build: &ports.ComposeBuild{
			Context: "../../src/uav_controller/ardupilot4_5_3",
			Dockerfile: "SITL",
			Args: map[string]string{
				"ARDUPILOT_BINARY_PATH": relPath,
			},
		},
	})

	appendServices := func(dir, category string, dockerfile string) {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					name := entry.Name()
					builder.AddService(ports.ComposeService{
						Name:  name,
						Image: prefixImage(name),
						Build: &ports.ComposeBuild{Context: fmt.Sprintf("../../src/%s/%s", category, name), Dockerfile: dockerfile},
					})
				}
			}
		}
	}

	appendServices(r.algorithmsDir, "algorithms", "Dockerfile")
	appendServices(r.mixersDir, "mixers", "Dockerfile")

	resDir := filepath.Join(simDir, "resources")
	os.MkdirAll(resDir, 0755)

	composePath := filepath.Join(simDir, "docker-compose.build.yaml")
	if err := os.WriteFile(composePath, []byte(builder.Build()), 0644); err != nil {
		return fmt.Errorf("failed to write build compose file: %w", err)
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

func (r *DockerRuntime) StartKubernetes(manifestPath, dockerHubUser string) error {
	r.ui.EmitEvent("simulation:log", "Kubernetes deployment logic not yet implemented. The manifest has been generated at: "+manifestPath)
	return nil
}

func (r *DockerRuntime) StopKubernetes(simName string) error {
	r.ui.EmitEvent("simulation:log", "Kubernetes stop logic not yet implemented.")
	return nil
}

func (r *DockerRuntime) CollectKubernetesLogs(simName, destDir string) error {
	r.ui.EmitEvent("simulation:log", "Kubernetes log collection not yet implemented.")
	return nil
}
