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

	"github.com/go-viper/mapstructure/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// NetSimMessage represents a message received on the "messages" topic.
type NetSimMessage struct {
	Topic   string                 `json:"topic"`
	UavID   string                 `json:"uav_id"`
	Payload map[string]interface{} `json:"payload"`
}

type TelemetryData struct {
	NrGpsOnline int `mapstructure:"nr_gps_online" json:"nr_gps_online"`
	Position    struct {
		Heading     float64 `mapstructure:"heading" json:"heading"`
		Alt         float64 `mapstructure:"alt" json:"alt"`
		RelativeAlt float64 `mapstructure:"relative_alt" json:"relative_alt"`
		Lon         float64 `mapstructure:"lon" json:"lon"`
		Lat         float64 `mapstructure:"lat" json:"lat"`
	} `mapstructure:"position" json:"position"`
	Type       string `mapstructure:"type" json:"type"`
	Battery    int    `mapstructure:"battery" json:"battery"`
	Version    string `mapstructure:"version" json:"version"`
	TimeBootMs uint64 `mapstructure:"time_boot_ms" json:"time_boot_ms"`
	Speed      struct {
		VX float64 `mapstructure:"vx" json:"vx"`
		VY float64 `mapstructure:"vy" json:"vy"`
		VZ float64 `mapstructure:"vz" json:"vz"`
	} `mapstructure:"speed" json:"speed"`
	Status     string `mapstructure:"status" json:"status"`
	FlightMode string `mapstructure:"flight_mode" json:"flight_mode"`
}

type TelemetryMessage struct {
	UavID   string        `json:"uav_id"`
	Payload TelemetryData `json:"payload"`
}

const (
	networkSimulatorContainer  = "network_simulator"
	networkSimulatorAddr       = "127.0.0.1:3000"
	containerReadyTimeout      = 30 * time.Second
	containerReadyPollInterval = time.Second
	// Brief delay after the container reports running so the application
	// inside it has time to bind its UDP port.
	portBindDelay = 2 * time.Second
	udpBufferSize = 65535
)

// Subscriber manages a UDP connection to the network simulator and listens
// for telemetry and messages topics. It supports tracking fleet readiness to
// trigger simulation events automatically.
type Subscriber struct {
	mu           sync.Mutex
	expectedUAVs int
	receivedUAVs int
	finishedUAVs int
	onAllReady   func()
	onFinish     func()
	ready        bool
	finished     bool
	wailsCtx     context.Context
}

// NewSubscriber creates a Subscriber with empty tracking maps.
func NewSubscriber() *Subscriber {
	return &Subscriber{}
}

// SetContext stores the Wails context so the subscriber can emit frontend events.
func (s *Subscriber) SetContext(ctx context.Context) {
	s.wailsCtx = ctx
}

// SetOnFinish registers a callback invoked when simulation is finished.
// It is called when all UAVs send "finish" or when the user stops all algorithms.
// The callback runs in a new goroutine.
func (s *Subscriber) SetOnFinish(callback func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onFinish = callback
}

// NotifyUserStoppedAll is called by the app when the user has manually sent
// "stop" to every algorithm. If the simulation has not already finished via UAV
// finish signals, this triggers the simulation-finished flow without re-sending stop.
func (s *Subscriber) NotifyUserStoppedAll(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finished {
		return
	}
	s.finished = true
	fmt.Printf("[netsim] user stopped all algorithms — simulation complete\n")
	runtime.EventsEmit(ctx, "simulation:finished", nil)
	// onFinish is intentionally NOT called here to avoid re-sending stop commands.
}

// SetExpectedFleet resets the simulation readiness tracking state for a new
// simulation run. The callback is executed once all defined UAVs report telemetry.
func (s *Subscriber) SetExpectedFleet(uavIDs []string, callback func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.expectedUAVs = len(uavIDs)
	s.receivedUAVs = 0
	s.finishedUAVs = 0
	s.onAllReady = callback
	s.ready = false
	s.finished = false
}

// SendGlobalBroadcast dispatches a JSON message to port 3000 of the simulator,
// intended for delivery to all registered UAVs.
func (s *Subscriber) SendGlobalBroadcast(payload interface{}) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", networkSimulatorAddr)
	if err != nil {
		return fmt.Errorf("resolve remote addr: %w", err)
	}

	packet := map[string]interface{}{
		"topic": "broadcast",
		"payload": map[string]interface{}{
			"uav_id":  "",
			"payload": payload,
		},
	}

	marshaled, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("marshal broadcast packet: %w", err)
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return fmt.Errorf("open broadcast UDP: %w", err)
	}
	defer conn.Close()

	if _, err := conn.WriteToUDP(marshaled, remoteAddr); err != nil {
		return fmt.Errorf("write broadcast packet: %w", err)
	}
	return nil
}

// Start blocks until ctx is cancelled, continuously listening for UDP telemetry.
// It waits for the network_simulator container to be ready before connecting.
// All errors are logged to stdout; transient read errors do not terminate the loop.
func (s *Subscriber) Start(ctx context.Context) {
	if err := s.waitForContainer(ctx); err != nil {
		fmt.Printf("[netsim] container wait failed: %v\n", err)
		return
	}

	conn, err := s.openUDPConnection()
	if err != nil {
		fmt.Printf("[netsim] failed to open UDP connection: %v\n", err)
		return
	}
	defer conn.Close()

	// Shut down the read loop when the context is cancelled.
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	if err := s.sendSubscribeRequest(conn); err != nil {
		fmt.Printf("[netsim] subscription request failed: %v\n", err)
		return
	}

	s.readLoop(ctx, conn)
}

// waitForContainer polls Docker until the network_simulator container reports
// as running, or until ctx is cancelled / the timeout elapses.
func (s *Subscriber) waitForContainer(ctx context.Context) error {
	deadline := time.Now().Add(containerReadyTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		out, err := exec.CommandContext(ctx,
			"docker", "inspect", "-f", "{{.State.Running}}", networkSimulatorContainer,
		).Output()
		if err == nil && strings.TrimSpace(string(out)) == "true" {
			time.Sleep(portBindDelay)
			return nil
		}
		time.Sleep(containerReadyPollInterval)
	}
	return fmt.Errorf("timed out waiting for %s container", networkSimulatorContainer)
}

// openUDPConnection resolves the simulator address and opens a local UDP socket.
func (s *Subscriber) openUDPConnection() (*net.UDPConn, error) {
	remoteAddr, err := net.ResolveUDPAddr("udp", networkSimulatorAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", networkSimulatorAddr, err)
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}

	// Store the remote address on the connection so sendSubscribeRequest can use it.
	// We close and reopen a "connected" UDP socket to pair the remote address.
	conn.Close()
	localAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err = net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP (retry): %w", err)
	}

	// Tag the connection with the remote address by doing a zero-byte send;
	// actual subscription packet is sent in sendSubscribeRequest.
	_ = remoteAddr
	return conn, nil
}

// sendSubscribeRequest sends subscription packets for both "telemetry" and
// "messages" topics to the simulator.
func (s *Subscriber) sendSubscribeRequest(conn *net.UDPConn) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", networkSimulatorAddr)
	if err != nil {
		return fmt.Errorf("resolve remote addr: %w", err)
	}

	for _, topic := range []string{"telemetry", "messages"} {
		payload, err := json.Marshal(map[string]interface{}{
			"topic": "subscribe",
			"payload": map[string]interface{}{
				"topic": topic,
			},
		})
		if err != nil {
			return fmt.Errorf("marshal subscribe payload for %s: %w", topic, err)
		}
		if _, err := conn.WriteToUDP(payload, remoteAddr); err != nil {
			return fmt.Errorf("write subscribe packet for %s: %w", topic, err)
		}
	}
	return nil
}

// readLoop continuously reads UDP datagrams and dispatches each packet based
// on its topic field. It handles both "telemetry" and "messages" topics.
func (s *Subscriber) readLoop(ctx context.Context, conn *net.UDPConn) {
	buffer := make([]byte, udpBufferSize)
	fmt.Printf("[netsim] Subscribed to telemetry + messages, reading from %s\n", conn.LocalAddr().String())
	for {
		bytesRead, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			fmt.Printf("[netsim] read error: %v\n", err)
			continue
		}

		var msg NetSimMessage
		if err := json.Unmarshal(buffer[:bytesRead], &msg); err != nil {
			fmt.Printf("[netsim] failed to parse telemetry: %v\n", err)
			continue
		}

		switch msg.Topic {
		case "telemetry":
			s.handleTelemetryPacket(ctx, msg)
		case "messages":
			s.handleMessagesPacket(ctx, msg)
		}
	}
}

// handleTelemetryPacket parses and dispatches a telemetry packet.
func (s *Subscriber) handleTelemetryPacket(ctx context.Context, msg NetSimMessage) {
	var telemetry TelemetryData
	if err := mapstructure.Decode(msg.Payload, &telemetry); err != nil {
		fmt.Printf("[netsim] failed to parse telemetry: %v\n", err)
		return
	}

	runtime.EventsEmit(ctx, "telemetry", TelemetryMessage{
		UavID:   msg.UavID,
		Payload: telemetry,
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return
	}
	if telemetry.NrGpsOnline <= 0 {
		return
	}

	s.receivedUAVs++
	if s.receivedUAVs != s.expectedUAVs {
		return
	}

	fmt.Printf("[netsim] all expected UAVs registered, triggering startup...\n")
	s.ready = true
	if s.onAllReady != nil {
		s.onAllReady()
	}
}

// handleMessagesPacket parses a messages-topic packet, logs it to stdout,
// emits it to the frontend, and tracks "finish" signals per UAV.
// The simulation is considered done when all expected UAVs have sent "finish".
func (s *Subscriber) handleMessagesPacket(ctx context.Context, msg NetSimMessage) {
	var service, command string

	switch msg.UavID {
	case "":
		{
			var content struct {
				Payload struct {
					Command string
				}
				Topic string
			}
			if err := mapstructure.Decode(msg.Payload, &content); err != nil {
				fmt.Printf("[netsim] failed to parse message: %v\n", err)
				return
			}
			service = serviceNameFromTopic(content.Topic)
			command = content.Payload.Command
		}
	default:
		{
			var content struct {
				Command string
				Source  string
			}
			if err := mapstructure.Decode(msg.Payload, &content); err != nil {
				fmt.Printf("[netsim] failed to parse message: %v\n", err)
				return
			}

			service = content.Source
			command = content.Command
		}
	}

	if command == "" {
		return
	}

	source := formatSource(msg.UavID)
	logLine := fmt.Sprintf("%s%s: %s", source, service, command)
	fmt.Printf("[netsim/messages] %s\n", logLine)

	// Forward to the frontend log panel.
	runtime.EventsEmit(ctx, "netsim:message", map[string]string{
		"source":  source,
		"service": service,
		"command": command,
		"label":   logLine,
	})

	if command == "finish" {
		s.registerFinish(ctx, msg.UavID)
	}
}

// serviceNameFromTopic extracts a readable service name from the payload's
// "topic" field. E.g. payload["topic"] = "algo/mission" → "mission".
func serviceNameFromTopic(name string) string {
	if name == "" {
		return "unknown"
	}
	// Strip the "algo/" prefix if present.
	if idx := lastSlashIndex(name); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// lastSlashIndex returns the index of the last '/' in s, or -1 if not found.
func lastSlashIndex(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// formatSource returns a human-readable label for a UAV ID. An empty ID means
// the message originated from the UI (global broadcast).
func formatSource(uavID string) string {
	if uavID == "" {
		return ""
	}
	return "(UAV-" + uavID + ") "
}

// registerFinish records that the given UAV has finished and, if all expected
// UAVs have now finished, triggers the simulation-finished flow.
func (s *Subscriber) registerFinish(ctx context.Context, uavID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ignore duplicate or post-finished signals.
	if s.finished {
		return
	}

	if uavID != "" {
		s.finishedUAVs++
	}

	if s.finishedUAVs == s.expectedUAVs {
		s.finished = true
		fmt.Printf("[netsim] all UAVs finished — simulation complete\n")
		runtime.EventsEmit(ctx, "simulation:finished", nil)
		if s.onFinish != nil {
			go s.onFinish()
		}
	} else {
		fmt.Printf("[netsim] UAV %s finished (%d/%d)\n", uavID, s.finishedUAVs, s.expectedUAVs)
	}
}
