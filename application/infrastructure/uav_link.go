package infrastructure

import (
	"encoding/json"
	"fmt"
	"net"

	"application/domain"
)

type DirectUAVLink struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func NewDirectUAVLink(ip string, port int) (*DirectUAVLink, error) {
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

	return &DirectUAVLink{
		conn: conn,
		addr: serverAddr,
	}, nil
}

func (l *DirectUAVLink) SendSuggestion(s domain.Suggestion) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	
	_, err = l.conn.WriteToUDP(data, l.addr)
	return err
}

func (l *DirectUAVLink) Close() error {
	if l.conn != nil {
		return l.conn.Close()
	}
	return nil
}
