package mysql

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/hsyan2008/go-logger/logger"
)

type Packet struct {
	Sequence uint8
	Conn     net.Conn

	w *bufio.Writer
}

func NewPacket(conn net.Conn) *Packet {
	// logger.Debug("NewPacket")
	return &Packet{
		Conn: conn,
		w:    bufio.NewWriter(conn),
	}
}

//负责读取除长度和序号外的数据部分
func (p *Packet) Read() (rs []byte, err error) {
	// logger.Debug("Packet Read")

	var buf = make([]byte, 4)
	_, _ = p.Conn.Read(buf)

	//get sequence
	sequence := uint8(buf[3])
	// logger.Debug("序号：", sequence)
	if sequence != p.Sequence {
		logger.Warnf("% x", buf)
		return nil, fmt.Errorf("invalid sequence recv %d != expect %d", sequence, p.Sequence)
	}
	p.Sequence++

	//get length
	var length uint32
	buf[3] = 0 //置为0，不影响下一行操作
	_ = binary.Read(bytes.NewReader(buf), binary.LittleEndian, &length)
	// logger.Debug("playload length：", length)

	rs = make([]byte, length)
	_, err = p.Conn.Read(rs)
	// logger.Warn(err)

	if length < MaxPayloadLen {
		return
	} else {
		buf, err = p.Read()
		if err != nil {
			// logger.Warn(err)
			return nil, err
		}
		return append(rs, buf...), nil
	}

	return
}

//发送数据，前面加上长度和序号，然后写入
func (p *Packet) Write(data []byte) (err error) {
	length := uint32(len(data))
	// logger.Debug("Packet Write length", length)
	var pos uint32 = 0
	if length >= MaxPayloadLen {
		_, err = p.w.Write([]byte{
			0xff,
			0xff,
			0xff,
			byte(p.Sequence),
		})
		_, _ = p.w.Write(data[pos : pos+MaxPayloadLen])
		_ = p.w.Flush()
		p.Sequence += 1
		length -= MaxPayloadLen
		pos += MaxPayloadLen
	}

	_, err = p.w.Write([]byte{
		byte(length),
		byte(length >> 8),
		byte(length >> 16),
		byte(p.Sequence),
	})
	_, err = p.w.Write(data[pos:])
	_ = p.w.Flush()
	p.Sequence += 1

	return
}

//发送数据，前面加上长度和序号，然后写入
//不能直接用p.Conn.Write，会被拆包
func (p *Packet) Write2(data []byte) (err error) {
	length := uint32(len(data))
	// logger.Debug("Packet Write length", length)
	var pos uint32 = 0
	if length >= MaxPayloadLen {
		buf := make([]byte, 4, 4+MaxPayloadLen)
		buf[0] = 0xff
		buf[1] = 0xff
		buf[2] = 0xff
		buf[3] = byte(p.Sequence)
		buf = append(buf, data[pos:pos+MaxPayloadLen]...)
		_, _ = p.Conn.Write(buf)
		p.Sequence += 1
		length -= MaxPayloadLen
		pos += MaxPayloadLen
	}

	buf := make([]byte, 4+length)
	buf[0] = byte(length)
	buf[1] = byte(length >> 8)
	buf[2] = byte(length >> 16)
	buf[3] = byte(p.Sequence)
	buf = append(buf, data[pos:]...)
	_, err = p.Conn.Write(buf)
	p.Sequence += 1

	return
}
