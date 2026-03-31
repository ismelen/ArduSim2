package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

type BrokerMessage struct {
	Topic   string      `json:"topic"`
	Payload interface{} `json:"payload"`
}

type UDPBroker struct {
	conn     *net.UDPConn
	handlers map[string][]func([]byte)
	hMu      sync.RWMutex
	stopChan chan struct{}
}

func NewUDPBroker() *UDPBroker {
	return &UDPBroker{
		handlers: make(map[string][]func([]byte)),
		stopChan: make(chan struct{}),
	}
}

func (b *UDPBroker) Connect(address string) error {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	b.conn = conn

	// Start background listener
	go b.listenLoop()

	return nil
}

func (b *UDPBroker) listenLoop() {
	buf := make([]byte, 65535)
	for {
		select {
		case <-b.stopChan:
			return
		default:
			n, _, err := b.conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}

			var msg BrokerMessage
			if err := json.Unmarshal(buf[:n], &msg); err == nil {
				b.hMu.RLock()
				handlers, ok := b.handlers[msg.Topic]
				b.hMu.RUnlock()

				if ok {
					payloadBytes, _ := json.Marshal(msg.Payload)
					for _, h := range handlers {
						go h(payloadBytes)
					}
				}
			}
		}
	}
}

func (b *UDPBroker) Publish(topic string, payload []byte) error {
	if b.conn == nil {
		return fmt.Errorf("not connected")
	}

	var payloadObj interface{}
	if err := json.Unmarshal(payload, &payloadObj); err != nil {
		return err
	}

	msg := BrokerMessage{
		Topic:   topic,
		Payload: payloadObj,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = b.conn.Write(data)
	return err
}

func (b *UDPBroker) Subscribe(topic string, handler func([]byte)) error {
	if b.conn == nil {
		return fmt.Errorf("not connected")
	}

	// 1. Send subscription command to broker
	subCmd := fmt.Sprintf("$subscribe %s", topic)
	_, err := b.conn.Write([]byte(subCmd))
	if err != nil {
		return err
	}

	// 2. Register handler localy
	b.hMu.Lock()
	b.handlers[topic] = append(b.handlers[topic], handler)
	b.hMu.Unlock()

	log.Printf("Subscribed to topic: %s", topic)
	return nil
}

func (b *UDPBroker) Close() error {
	if b.conn != nil {
		close(b.stopChan)
		return b.conn.Close()
	}
	return nil
}
