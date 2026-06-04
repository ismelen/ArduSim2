package input

import "net"

type RawPacket struct {
	Data []byte
	Addr *net.UDPAddr
}

type MessageHandler interface {
	Handle(pkt RawPacket)
}
