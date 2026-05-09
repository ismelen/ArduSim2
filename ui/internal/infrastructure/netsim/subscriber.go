package netsim

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"ui/internal/domain"
	"ui/internal/ports"

	"github.com/go-viper/mapstructure/v2"
)


type NetsimSubscriber struct {
	mu           sync.Mutex
	ui           ports.UIBridge
	expectedUAVs int
	receivedUAVs map[string]bool
	finishedUAVs map[string]bool
	onFinish     func()
	ready        bool
	finished     bool
	remoteAddr   string
	useDockerWait bool
}

func NewNetsimSubscriber(ui ports.UIBridge) *NetsimSubscriber {
	return &NetsimSubscriber{
		ui:            ui,
		remoteAddr:    "127.0.0.1:3000",
		useDockerWait: true,
	}
}

func (s *NetsimSubscriber) SetRemoteAddr(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteAddr = ip + ":3000"
	s.useDockerWait = false
}

func (s *NetsimSubscriber) SetExpectedFleet(uavIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expectedUAVs = len(uavIDs)
	s.receivedUAVs = make(map[string]bool)
	s.finishedUAVs = make(map[string]bool)
	s.ready = false
	s.finished = false
}

func (s *NetsimSubscriber) SetOnFinish(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onFinish = fn
}

func (s *NetsimSubscriber) SendGlobalBroadcast(payload interface{}) error {
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", s.remoteAddr)
	if err != nil {
		return err
	}

	packet := map[string]interface{}{
		"topic": "broadcast",
		"payload": map[string]interface{}{
			"uav_id":  "",
			"payload": payload,
		},
	}

	marshaled, _ := json.Marshal(packet)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.WriteToUDP(marshaled, remoteUDPAddr)
	return err
}

func (s *NetsimSubscriber) NotifyUserStoppedAll(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finished {
		return
	}
	s.finished = true
	s.ui.EmitEvent("simulation:finished", nil)
}

func (s *NetsimSubscriber) Start(ctx context.Context) {
	if s.useDockerWait {
		_ = s.waitForContainer(ctx)
	}

	remoteUDPAddr, _ := net.ResolveUDPAddr("udp", s.remoteAddr)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	// Subscribe in a loop until we get a message or timeout
	go func() {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			for _, topic := range []string{"telemetry", "messages"} {
				p, _ := json.Marshal(map[string]interface{}{
					"topic": "subscribe",
					"payload": map[string]interface{}{"topic": topic},
				})
				_, _ = conn.WriteToUDP(p, remoteUDPAddr)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	buffer := make([]byte, 65535)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return
		}

		var msg struct {
			Topic   string                 `json:"topic"`
			UavID   string                 `json:"uav_id"`
			Payload map[string]interface{} `json:"payload"`
		}
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}

		switch msg.Topic {
		case "telemetry":
			var t domain.TelemetryData
			_ = mapstructure.Decode(msg.Payload, &t)
			if t.NrGpsOnline == 0 { continue }
			
			s.ui.EmitEvent("telemetry", map[string]interface{}{
				"uav_id":  msg.UavID,
				"payload": t,
			})

			s.mu.Lock()
			if !s.ready {
				s.receivedUAVs[msg.UavID] = true
				keys := make([]string, 0, len(s.receivedUAVs))
				for k := range s.receivedUAVs {
					keys = append(keys, k)
				}
				if len(keys) == s.expectedUAVs {
					s.ready = true
					s.ui.EmitEvent("netsim:message", map[string]string{
						"label": "All Ready",
					})
				}
			}
			s.mu.Unlock()
		case "messages":
			s.handleMessagesPacket(msg)
		}
	}
}

func (s *NetsimSubscriber) handleMessagesPacket(msg struct {
	Topic   string                 `json:"topic"`
	UavID   string                 `json:"uav_id"`
	Payload map[string]interface{} `json:"payload"`
}) {
	var service, command string
	if msg.UavID == "" {
		var content struct {
			Payload struct{ Command string }
			Topic   string
		}
		_ = mapstructure.Decode(msg.Payload, &content)
		service = s.serviceNameFromTopic(content.Topic)
		command = content.Payload.Command
	} else {
		var content struct {
			Command string
			Source  string
		}
		_ = mapstructure.Decode(msg.Payload, &content)
		service = content.Source
		command = content.Command
	}

	if command == "" {
		return
	}

	source := s.formatSource(msg.UavID)
	logLine := fmt.Sprintf("%s%s: %s", source, service, command)

	s.ui.EmitEvent("netsim:message", map[string]string{
		"source":  source,
		"label":   logLine,
	})

	if command == "finish" {
		s.registerFinish(msg.UavID)
	}
}

func (s *NetsimSubscriber) serviceNameFromTopic(name string) string {
	if name == "" {
		return "unknown"
	}
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

func (s *NetsimSubscriber) formatSource(uavID string) string {
	if uavID == "" {
		return ""
	}
	return "(UAV-" + uavID + ") "
}

func (s *NetsimSubscriber) registerFinish(uavID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finished {
		return
	}
	if uavID != "" {
		s.finishedUAVs[uavID] = true
	}
	keys := make([]string, 0, len(s.finishedUAVs))
	for k := range s.finishedUAVs {
		keys = append(keys, k)
	}
	if len(keys) == s.expectedUAVs {
		s.finished = true
		s.ui.EmitEvent("simulation:finished", nil)
		if s.onFinish != nil {
			go s.onFinish()
		}
	}
}

func (s *NetsimSubscriber) waitForContainer(ctx context.Context) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", "network_simulator").Output()
		if err == nil && strings.TrimSpace(string(out)) == "true" {
			time.Sleep(2 * time.Second)
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout")
}
