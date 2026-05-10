package udp

import (
	"net"
	"netsim/ports/input"
	"netsim/ports/output"
)

type Receiver struct {
	port   int
	logger output.Logger
}

func NewReceiver(port int, logger output.Logger) *Receiver {
	return &Receiver{port: port, logger: logger}
}

func (r *Receiver) Run(handler input.MessageHandler) {
	addr := &net.UDPAddr{Port: r.port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		r.logger.Error("Failed to start UDP receiver", "error", err)
		return
	}
	defer conn.Close()

	r.logger.Info("UDP Receiver started", "port", r.port)
	buf := make([]byte, 65507)

	for {
		n, peerAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			r.logger.Error("Error reading UDP", "error", err)
			continue
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		handler.Handle(input.RawPacket{
			Data: data,
			Addr: peerAddr,
		})
	}
}
