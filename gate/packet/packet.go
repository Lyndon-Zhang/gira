package packet

import (
	"errors"
	"fmt"
)

type Type byte

const (
	_              Type = iota
	TypeMin             = Handshake
	Handshake           = 0x01
	HandshakeAck        = 0x02
	Data                = 0x03
	Heartbeat           = 0x04
	Kick                = 0x05
	ServerDown          = 0x06
	ServerMaintain      = 0x07
	UserInstead         = 0x08
	ServerSuspend       = 0x09
	ServerResume        = 0x0a
	TypeMax             = ServerResume
)

var ErrWrongPacketType = errors.New("wrong packet type")

type Packet struct {
	Type   Type
	Length int
	Data   []byte
}

func NewPacket() *Packet {
	return &Packet{}
}

func (p *Packet) String() string {
	return fmt.Sprintf("Type: %d, Length: %d, Data: %s", p.Type, p.Length, string(p.Data))
}
