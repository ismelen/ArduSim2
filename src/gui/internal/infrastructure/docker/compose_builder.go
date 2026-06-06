package docker

import (
	"fmt"
	"strings"
)

// ComposeService represents a generic docker-compose service
type ComposeService struct {
	Name          string
	Image         string
	ContainerName string
	DependsOn     []string
	Environment   map[string]string
	Ports         []string
	Volumes       []string
	Networks      map[string][]string // map network name to list of aliases
	ExtraHosts    []string
	Limits        ResourceLimits
}

type composeBuilder struct {
	services strings.Builder
	networks strings.Builder
}

func newComposeBuilder() *composeBuilder {
	b := &composeBuilder{}
	b.services.WriteString("services:\n")
	b.networks.WriteString("networks:\n")
	return b
}

func (b *composeBuilder) AddService(svc ComposeService) {
	fmt.Fprintf(&b.services, "  %s:\n    image: %s\n", svc.Name, svc.Image)
	
	if svc.ContainerName != "" {
		fmt.Fprintf(&b.services, "    container_name: %s\n", svc.ContainerName)
	}

	if len(svc.DependsOn) > 0 {
		fmt.Fprintf(&b.services, "    depends_on:\n")
		for _, dep := range svc.DependsOn {
			fmt.Fprintf(&b.services, "      - %s\n", dep)
		}
	}

	if len(svc.ExtraHosts) > 0 {
		fmt.Fprintf(&b.services, "    extra_hosts:\n")
		for _, h := range svc.ExtraHosts {
			fmt.Fprintf(&b.services, "      - %q\n", h)
		}
	}

	if len(svc.Ports) > 0 {
		fmt.Fprintf(&b.services, "    ports:\n")
		for _, p := range svc.Ports {
			fmt.Fprintf(&b.services, "      - %s\n", p)
		}
	}

	if len(svc.Environment) > 0 {
		fmt.Fprintf(&b.services, "    environment:\n")
		for k, v := range svc.Environment {
			fmt.Fprintf(&b.services, "      - %s=%s\n", k, v)
		}
	}

	if len(svc.Volumes) > 0 {
		fmt.Fprintf(&b.services, "    volumes:\n")
		for _, v := range svc.Volumes {
			fmt.Fprintf(&b.services, "      - %s\n", v)
		}
	}

	if len(svc.Networks) > 0 {
		fmt.Fprintf(&b.services, "    networks:\n")
		for netName, aliases := range svc.Networks {
			fmt.Fprintf(&b.services, "      %s:\n", netName)
			if len(aliases) > 0 {
				fmt.Fprintf(&b.services, "        aliases:\n")
				for _, a := range aliases {
					fmt.Fprintf(&b.services, "          - %s\n", a)
				}
			}
		}
	}

	if svc.Limits.HasLimits() {
		if svc.Limits.RAM != "" {
			fmt.Fprintf(&b.services, "    mem_limit: %s\n", svc.Limits.RAM)
		}
		if svc.Limits.CPU > 0 {
			fmt.Fprintf(&b.services, "    cpus: \"%.2f\"\n", svc.Limits.CPU)
		}
	}
	fmt.Fprintf(&b.services, "\n")
}

// addNetwork appends a bridge network definition if not already present.
func (b *composeBuilder) addNetwork(name string, subnet string) {
	if subnet != "" {
		fmt.Fprintf(&b.networks, "  %s:\n    driver: bridge\n    ipam:\n      config:\n        - subnet: %s\n", name, subnet)
	} else {
		fmt.Fprintf(&b.networks, "  %s:\n    driver: bridge\n", name)
	}
}

func (b *composeBuilder) Build() string {
	return b.services.String() + "\n" + b.networks.String()
}
