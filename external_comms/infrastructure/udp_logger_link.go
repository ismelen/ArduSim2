package infrastructure

import (
	"encoding/json"
	"fmt"
	"net"
)

type UDPLoggerLink struct {
	conn *net.UDPConn
}

func NewUDPLoggerLink(ip string, port int) (*UDPLoggerLink, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	return &UDPLoggerLink{
		conn: conn,
	}, nil
}

func (u *UDPLoggerLink) SendLog(payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = u.conn.Write(data)
	return err
}

func (u *UDPLoggerLink) Close() error {
	if u.conn != nil {
		return u.conn.Close()
	}
	return nil
}
