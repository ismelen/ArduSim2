package docker

import (
	"fmt"
	"strings"
)

type kubernetesBuilder struct {
	manifests strings.Builder
}

func newKubernetesBuilder() *kubernetesBuilder {
	return &kubernetesBuilder{}
}

// AddConfigMap creates a ConfigMap for all files (JSON, params, KML, etc.)
func (b *kubernetesBuilder) AddConfigMap(name string, files map[string]string) {
	fmt.Fprintf(&b.manifests, "---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: %s\ndata:\n", name)
	for fileName, content := range files {
		fmt.Fprintf(&b.manifests, "  %s: |-\n", fileName)
		// Ensure proper indentation for multiline content
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if line == "" {
				fmt.Fprintf(&b.manifests, "\n")
			} else {
				fmt.Fprintf(&b.manifests, "    %s\n", line)
			}
		}
	}
	fmt.Fprintf(&b.manifests, "\n")
}

type KubeContainer struct {
	Name         string
	Image        string
	Ports        []KubePort
	Env          map[string]string
	VolumeMounts []KubeVolumeMount
}

type KubePort struct {
	ContainerPort int
	Protocol      string // UDP, TCP
}

type KubeVolumeMount struct {
	Name      string
	MountPath string
	SubPath   string
}

func (b *kubernetesBuilder) AddDeployment(name string, containers []KubeContainer, volumes []string) {
	fmt.Fprintf(&b.manifests, "---\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: %s\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: %s\n  template:\n    metadata:\n      labels:\n        app: %s\n    spec:\n      containers:\n", name, name, name)

	for _, c := range containers {
		fmt.Fprintf(&b.manifests, "      - name: %s\n        image: %s\n", c.Name, c.Image)
		
		if len(c.Ports) > 0 {
			fmt.Fprintf(&b.manifests, "        ports:\n")
			for _, p := range c.Ports {
				protocol := "TCP"
				if p.Protocol != "" {
					protocol = p.Protocol
				}
				fmt.Fprintf(&b.manifests, "        - containerPort: %d\n          protocol: %s\n", p.ContainerPort, protocol)
			}
		}

		if len(c.Env) > 0 {
			fmt.Fprintf(&b.manifests, "        env:\n")
			for k, v := range c.Env {
				// We don't quote blindly if it's already structured, but for simple strings it's safe.
				// However, if the value contains newlines, %q handles it.
				fmt.Fprintf(&b.manifests, "        - name: %s\n          value: %q\n", k, v)
			}
		}

		if len(c.VolumeMounts) > 0 {
			fmt.Fprintf(&b.manifests, "        volumeMounts:\n")
			for _, vm := range c.VolumeMounts {
				fmt.Fprintf(&b.manifests, "        - name: %s\n          mountPath: %s\n", vm.Name, vm.MountPath)
				if vm.SubPath != "" {
					fmt.Fprintf(&b.manifests, "          subPath: %s\n", vm.SubPath)
				}
			}
		}
	}

	if len(volumes) > 0 {
		fmt.Fprintf(&b.manifests, "      volumes:\n")
		for _, v := range volumes {
			fmt.Fprintf(&b.manifests, "      - name: %s\n        configMap:\n          name: %s\n", v, v)
		}
	}

	fmt.Fprintf(&b.manifests, "\n")
}

func (b *kubernetesBuilder) Build() string {
	return b.manifests.String()
}
