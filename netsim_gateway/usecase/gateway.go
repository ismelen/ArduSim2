package usecase

import (
	"encoding/json"
	"net"
	"netsim_gateway/domain/model"
	"netsim_gateway/domain/service"
	"netsim_gateway/ports/input"
	"netsim_gateway/ports/output"
	"sync"
)

type Config struct {
	UAVListenPort     int
	NetsimListenPort  int
	SnapshotIntervalS int
	NetsimDiscovery   DiscoveryConfig
}

type DiscoveryConfig struct {
	Mode                string
	Addrs               []string
	SwarmService        string
	RediscoverIntervalS int
}

type Gateway struct {
	Config         Config
	netsimAddrs    []*net.UDPAddr
	uavRegistry    sync.Map // string (uavID) -> *net.UDPAddr
	telemetryCache sync.Map // string (uavID) -> json.RawMessage
	uiSubscribers  sync.Map // string (addr.String()) -> *net.UDPAddr
	uavSender      output.PacketSender
	netsimSender   output.PacketSender
	logger         output.Logger
	mu             sync.RWMutex
}

func NewGateway(cfg Config, netsims []*net.UDPAddr, uavSender, netsimSender output.PacketSender, logger output.Logger) *Gateway {
	return &Gateway{
		Config:       cfg,
		netsimAddrs:  netsims,
		uavSender:    uavSender,
		netsimSender: netsimSender,
		logger:       logger,
	}
}

func (g *Gateway) UpdateNetsims(addrs []*net.UDPAddr) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.netsimAddrs = addrs
}

type InboundMsg struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

type TelemetryPayload struct {
	UAVID string `json:"uav_id"`
}

func (g *Gateway) HandleUAVMessage(pkt input.RawPacket) {
	var msg InboundMsg
	if err := json.Unmarshal(pkt.Data, &msg); err != nil {
		return
	}

	if msg.Topic == "subscribe" {
		g.uiSubscribers.Store(pkt.Addr.String(), pkt.Addr)
		g.logger.Info("New UI subscriber", "addr", pkt.Addr.String())
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(msg.Payload, &payload); err == nil {
		if uavID, ok := payload["uav_id"].(string); ok {
			if uavID != "" {
				g.uavRegistry.Store(uavID, pkt.Addr)
			}

			if msg.Topic == "broadcast" && uavID == "" {
				if data, err := json.Marshal(payload); err == nil {
					g.uavRegistry.Range(func(key, value any) bool {
						addr := value.(*net.UDPAddr)
						g.uavSender.Send(data, addr)
						return true
					})
				}
				return
			}

			g.mu.RLock()
			netsimCount := len(g.netsimAddrs)
			var target *net.UDPAddr
			if netsimCount > 0 {
				idx := service.ConsistentHash(uavID, netsimCount)
				target = g.netsimAddrs[idx]
			}
			g.mu.RUnlock()

			if target != nil {
				wrapped := map[string]any{
					"topic":   "uav_" + msg.Topic,
					"payload": payload,
				}
				if data, err := json.Marshal(wrapped); err == nil {
					g.netsimSender.Send(data, target)
				}
			}
		}
	}
}

type DeliverMessage struct {
	TargetUAVID string `json:"target_uav_id"`
	SenderID    string `json:"sender_id"`
	Payload     string `json:"payload"`
}

type BroadcastNotify struct {
	SenderID       string          `json:"sender_id"`
	SenderPosition map[string]any  `json:"sender_position"`
	Payload        json.RawMessage `json:"payload"`
}

func (g *Gateway) HandleNetsimMessage(pkt input.RawPacket) {
	var msg InboundMsg
	if err := json.Unmarshal(pkt.Data, &msg); err != nil {
		return
	}

	switch msg.Topic {
	case "snapshot":
		var snap model.TelemetrySnapshot
		if err := json.Unmarshal(msg.Payload, &snap); err == nil {
			for uavID, telemetry := range snap.UAVs {
				g.telemetryCache.Store(uavID, telemetry)
			}
		}
	case "deliver":
		var deliver DeliverMessage
		if err := json.Unmarshal(msg.Payload, &deliver); err == nil {
			if val, ok := g.uavRegistry.Load(deliver.TargetUAVID); ok {
				addr := val.(*net.UDPAddr)
				var parsedPayload map[string]interface{}
				if err := json.Unmarshal([]byte(deliver.Payload), &parsedPayload); err == nil {
					formatted := map[string]interface{}{
						"uav_id":  deliver.SenderID,
						"payload": parsedPayload,
					}
					data, _ := json.Marshal(formatted)
					g.uavSender.Send(data, addr)
				}
			}
		}
	case "broadcast_notify":
		var notify BroadcastNotify
		if err := json.Unmarshal(msg.Payload, &notify); err == nil {
			uiMsg := map[string]any{
				"topic": "broadcast",
				"payload": map[string]any{
					"sender_id": notify.SenderID,
					"payload":   notify.Payload,
				},
			}
			if uiData, err := json.Marshal(uiMsg); err == nil {
				g.uiSubscribers.Range(func(key, value any) bool {
					addr := value.(*net.UDPAddr)
					g.uavSender.Send(uiData, addr)
					return true
				})
			}

			g.mu.RLock()
			for _, netsimAddr := range g.netsimAddrs {
				if netsimAddr.String() != pkt.Addr.String() {
					peerMsg := map[string]any{
						"topic": "peer_broadcast",
						"payload": map[string]any{
							"sender_id":       notify.SenderID,
							"sender_position": notify.SenderPosition,
							"payload":         notify.Payload,
							"origin_node":     pkt.Addr.String(),
						},
					}
					if peerData, err := json.Marshal(peerMsg); err == nil {
						g.netsimSender.Send(peerData, netsimAddr)
					}
				}
			}
			g.mu.RUnlock()
		}
	}
}

type UAVHandler struct{ gateway *Gateway }

func (h *UAVHandler) Handle(pkt input.RawPacket) { h.gateway.HandleUAVMessage(pkt) }

type NetsimHandler struct{ gateway *Gateway }

func (h *NetsimHandler) Handle(pkt input.RawPacket) { h.gateway.HandleNetsimMessage(pkt) }

func NewUAVHandler(g *Gateway) input.UAVMessageHandler       { return &UAVHandler{gateway: g} }
func NewNetsimHandler(g *Gateway) input.NetsimMessageHandler { return &NetsimHandler{gateway: g} }
