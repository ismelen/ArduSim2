package infrastructure

import (
	"encoding/json"
	"fmt"
	"net"

	"external_comms/domain"
)

type UDPNetSimLink struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func NewUDPNetSimLink(ip string, port int) (*UDPNetSimLink, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, err
	}

	return &UDPNetSimLink{
		conn: conn,
		addr: serverAddr,
	}, nil
}

func (n *UDPNetSimLink) Send(msg domain.SendedNetSimMessage) error {
	valid_msg := map[string]interface{}{
		"topic": msg.Topic,
		"payload": map[string]interface{}{
			"payload":   msg.Payload,
			"uav_id": msg.Source,
		},
	}
	data, err := json.Marshal(valid_msg)
	if err != nil {
		return err
	}
	_, err = n.conn.WriteToUDP(data, n.addr)
	return err
}

func (n *UDPNetSimLink) Listen() (<-chan domain.ReceivedNetSimMessage, error) {
	msgChan := make(chan domain.ReceivedNetSimMessage, 100)

	go func() {
		defer close(msgChan)
		buffer := make([]byte, 4096)
		for {
			bytesRead, _, err := n.conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}

			var msg domain.ReceivedNetSimMessage
			if err := json.Unmarshal(buffer[:bytesRead], &msg); err == nil {
				msgChan <- msg
			} else {
				fmt.Println(err.Error())
			}
		}
	}()

	return msgChan, nil
}

func (n *UDPNetSimLink) Close() error {
	if n.conn != nil {
		return n.conn.Close()
	}
	return nil
}
