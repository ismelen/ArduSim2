package broker

import (
	"encoding/json"
	"fmt"
	"net"

	"mission/domain"
)

type UDPBroker struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func NewUDPBroker() *UDPBroker {
	return &UDPBroker{}
}

func (b *UDPBroker) Connect(ip string, port int, subTopic, telemetryTopic string) error {
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return err
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return err
	}

	b.conn = conn
	b.addr = serverAddr

	b.sendSubscription(subTopic)
	b.sendSubscription(telemetryTopic)

	return nil
}

func (b *UDPBroker) sendSubscription(topic string) {
	msg := map[string]interface{}{
		"topic":        "$subscribe",
		"subscribe_to": topic,
	}
	b.publishJSON(msg)
}

func (b *UDPBroker) Publish(topic string, payload map[string]interface{}) error {
	payload["topic"] = topic
	return b.publishJSON(payload)
}

func (b *UDPBroker) publishJSON(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = b.conn.WriteToUDP(data, b.addr)
	return err
}

func (b *UDPBroker) Listen() (<-chan domain.BrokerMessage, error) {
	msgChan := make(chan domain.BrokerMessage)

	go func() {
		defer close(msgChan)
		buffer := make([]byte, 4096)
		for {
			n, _, err := b.conn.ReadFromUDP(buffer)
			if err != nil {
				return // Connection closed
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(buffer[:n], &payload); err == nil {
				if topic, ok := payload["topic"].(string); ok {
					msgChan <- domain.BrokerMessage{
						Topic:   topic,
						Payload: payload,
					}
				}
			}
		}
	}()

	return msgChan, nil
}

func (b *UDPBroker) Close() error {
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}
