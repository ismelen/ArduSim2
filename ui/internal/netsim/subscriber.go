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

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type TelemetryPosition struct {
	Heading     float64 `json:"heading"`
	Alt         float64 `json:"alt"`
	RelativeAlt float64 `json:"relative_alt"`
	Lon         float64 `json:"lon"`
	Lat         float64 `json:"lat"`
}

type TelemetrySpeed struct {
	VX float64 `json:"vx"`
	VY float64 `json:"vy"`
	VZ float64 `json:"vz"`
}

type TelemetryData struct {
	NrGpsOnline int               `json:"nr_gps_online"`
	Position    TelemetryPosition `json:"position"`
	Type        string            `json:"type"`
	Battery     int               `json:"battery"`
	Version     string            `json:"version"`
	TimeBootMs  uint64            `json:"time_boot_ms"`
	Speed       TelemetrySpeed    `json:"speed"`
	Status      string            `json:"status"`
	FlightMode  string            `json:"flight_mode"`
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
// for telemetry messages. It supports tracking fleet readiness to trigger
// simulation events automatically.
type Subscriber struct {
	mu           sync.Mutex
	expectedUAVs map[string]bool
	receivedUAVs map[string]bool
	onAllReady   func()
	ready        bool
}

// NewSubscriber creates a Subscriber with empty tracking maps.
func NewSubscriber() *Subscriber {
	return &Subscriber{
		expectedUAVs: make(map[string]bool),
		receivedUAVs: make(map[string]bool),
	}
}

// SetExpectedFleet resets the simulation readiness tracking state for a new
// simulation run. The callback is executed once all defined UAVs report telemetry.
func (s *Subscriber) SetExpectedFleet(uavIDs []string, callback func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.expectedUAVs = make(map[string]bool)
	s.receivedUAVs = make(map[string]bool)
	s.onAllReady = callback
	s.ready = false

	for _, id := range uavIDs {
		s.expectedUAVs[id] = true
	}
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

// sendSubscribeRequest sends the JSON subscription packet to the simulator.
func (s *Subscriber) sendSubscribeRequest(conn *net.UDPConn) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", networkSimulatorAddr)
	if err != nil {
		return fmt.Errorf("resolve remote addr: %w", err)
	}

	payload, err := json.Marshal(map[string]interface{}{
		"topic": "subscribe",
	})
	if err != nil {
		return fmt.Errorf("marshal subscribe payload: %w", err)
	}

	if _, err := conn.WriteToUDP(payload, remoteAddr); err != nil {
		return fmt.Errorf("write subscribe packet: %w", err)
	}
	return nil
}

// readLoop continuously reads UDP datagrams, parses them as TelemetryMessage,
// and emits them as Wails events until the connection is closed.
func (s *Subscriber) readLoop(ctx context.Context, conn *net.UDPConn) {
	buffer := make([]byte, udpBufferSize)
	fmt.Printf("[netsim] Reading from %s\n", conn.LocalAddr().String())
	for {
		bytesRead, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			fmt.Printf("[netsim] read error: %v\n", err)
			continue
		}

		var msg TelemetryMessage
		if err := json.Unmarshal(buffer[:bytesRead], &msg); err != nil {
			// Log error but continue listening for next packet.
			fmt.Printf("[netsim] failed to parse telemetry: %v\n", err)
			continue
		}

		// Emit the telemetry event to the frontend.
		runtime.EventsEmit(ctx, "telemetry", msg)

		// Check-in logic to trigger automatic mission start.
		s.mu.Lock()
		if !s.ready && s.expectedUAVs[msg.UavID] {
			s.receivedUAVs[msg.UavID] = true
			if len(s.receivedUAVs) == len(s.expectedUAVs) {
				s.ready = true
				fmt.Printf("[netsim] all expected UAVs registered, triggering startup...\n")
				if s.onAllReady != nil {
					go s.onAllReady()
				}
			}
		}
		s.mu.Unlock()
	}
}
