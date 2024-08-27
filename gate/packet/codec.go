package packet

import (
	"bytes"
	"errors"
)

const (
	HEAD_LENGTH     = 4
	MAX_PACKET_SIZE = 64 * 1024
)

var ErrPacketSizeExcced = errors.New("codec: packet size exceed")

type Decoder struct {
	buf  *bytes.Buffer
	size int  // last packet length
	typ  byte // last packet type
}

func NewDecoder() *Decoder {
	return &Decoder{
		buf:  bytes.NewBuffer(nil),
		size: -1,
	}
}

func (c *Decoder) forward() error {
	header := c.buf.Next(HEAD_LENGTH)
	c.typ = header[0]
	if c.typ < TypeMin || c.typ > TypeMax {
		return ErrWrongPacketType
	}
	c.size = bytesToInt(header[1:])
	if c.size > MAX_PACKET_SIZE {
		return ErrPacketSizeExcced
	}
	return nil
}

func (c *Decoder) Decode(data []byte) ([]*Packet, error) {
	c.buf.Write(data)
	var (
		packets []*Packet
		err     error
	)
	if c.buf.Len() < HEAD_LENGTH {
		return nil, err
	}
	if c.size < 0 {
		if err = c.forward(); err != nil {
			return nil, err
		}
	}
	for c.size <= c.buf.Len() {
		p := &Packet{Type: Type(c.typ), Length: c.size, Data: c.buf.Next(c.size)}
		packets = append(packets, p)
		if c.buf.Len() < HEAD_LENGTH {
			c.size = -1
			break
		}
		if err = c.forward(); err != nil {
			return nil, err
		}
	}
	return packets, nil
}

// -<type>-|--------<length>--------|-<data>-
func Encode(typ Type, data []byte) ([]byte, error) {
	if typ < TypeMin || typ > TypeMax {
		return nil, ErrWrongPacketType
	}
	p := &Packet{Type: typ, Length: len(data)}
	buf := make([]byte, p.Length+HEAD_LENGTH)
	buf[0] = byte(p.Type)
	copy(buf[1:HEAD_LENGTH], intToBytes(p.Length))
	copy(buf[HEAD_LENGTH:], data)
	return buf, nil
}

func bytesToInt(b []byte) int {
	result := 0
	for _, v := range b {
		result = result<<8 + int(v)
	}
	return result
}

func intToBytes(n int) []byte {
	buf := make([]byte, 3)
	buf[0] = byte((n >> 16) & 0xFF)
	buf[1] = byte((n >> 8) & 0xFF)
	buf[2] = byte(n & 0xFF)
	return buf
}
