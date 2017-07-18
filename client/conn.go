package client

//用于发起连接

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	. "github.com/hsyan2008/sunder/mysql"
)

type Conn struct {
	p *Packet

	mysqlVer   string
	connectId  uint32
	salt       []byte
	capability uint32
	status     uint16

	collationId CollationId

	authPlugin string

	username string
	password string
	database string

	isHandshake  bool
	lastPingTime int64

	Kind Kind
}

type Kind uint8

const (
	WRITER Kind = iota
	READER
)

func NewConn(addr, username, password, database string) (*Conn, error) {

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}

	return &Conn{
		p:            NewPacket(conn),
		username:     username,
		password:     password,
		database:     database,
		lastPingTime: time.Now().Unix(),
	}, nil
}

//临时  TODO
func (c *Conn) GetConn() net.Conn {
	if c.isHandshake == false {
		_ = c.Handshake()
	}
	return c.p.Conn
}

func (c *Conn) GetLastPingTime() int64 {
	return c.lastPingTime
}

func (c *Conn) Handshake() (err error) {
	// logger.Debug("client Handshake")
	err = c.ReadHandshake()
	if err != nil {
		return
	}
	err = c.WriteHandshakeResponse()
	if err != nil {
		return
	}
	_, err = c.readOK()
	if err != nil {
		return
	}

	c.isHandshake = true
	//握手完毕，重新开始计算序号
	c.p.Sequence = 0

	return
}

func (c *Conn) ReadHandshake() error {
	// logger.Debug("client ReadHandshake")
	data, err := c.p.Read()

	if err != nil {
		return err
	}

	pos := 0

	if data[pos] == ERR_HEADER {
		return errors.New("read initial handshake error")
	}

	if data[pos] < MinProtocolVersion {
		return fmt.Errorf("invalid protocol version %d, must >= 10", data[0])
	}

	//skip protocol version
	pos += 1

	verPos := bytes.IndexByte(data[pos:], 0x00)
	c.mysqlVer = string(data[pos : verPos+1])
	pos += verPos + 1

	c.connectId = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4

	c.salt = append(c.salt, data[pos:pos+8]...)
	pos += 8

	//skip filter
	pos += 1

	c.capability = uint32(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2

	c.collationId = CollationId(data[pos])
	pos += 1

	c.status = uint16(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2

	c.capability = uint32(binary.LittleEndian.Uint16(data[pos:pos+2]))<<16 | c.capability
	pos += 2

	//skip auth data len or filter
	pos += 1
	//skip reserved
	pos += 10

	c.salt = append(c.salt, data[pos:pos+12]...)
	//include 0x00
	pos += 12 + 1

	if pos < len(data) {
		c.authPlugin = string(data[pos : len(data)-1])
	}

	// logger.Debugf("客户端解析Handshake数据：% x", data)
	// logger.Debug("客户端解析Handshake结果：", *c)

	return err
}

func (c *Conn) WriteHandshakeResponse() error {
	// logger.Debug("client WriteHandshakeResponse")
	// Adjust client capability flags based on server support
	capability := CLIENT_PROTOCOL_41 | CLIENT_SECURE_CONNECTION |
		CLIENT_LONG_PASSWORD | CLIENT_TRANSACTIONS | CLIENT_LONG_FLAG

	capability &= c.capability

	if len(c.database) > 0 {
		capability |= CLIENT_CONNECT_WITH_DB
	}
	if len(c.authPlugin) > 0 {
		capability |= CLIENT_PLUGIN_AUTH
	}
	c.capability = capability

	data := bytes.NewBuffer([]byte{
		byte(c.capability),
		byte(c.capability >> 8),
		byte(c.capability >> 16),
		byte(c.capability >> 24),
	})

	//最大数据包长度
	_, _ = data.Write(make([]byte, 4))

	//字符集
	_ = data.WriteByte(byte(c.collationId))

	//填充23个字节
	_, _ = data.Write(make([]byte, 23))

	//用户名和空字符
	_, _ = data.Write([]byte(c.username + "\x00"))

	//密码长度
	auth := Sha1Password(c.password, c.salt)
	_ = data.WriteByte(byte(len(auth)))
	_, _ = data.Write(auth)

	//db
	if len(c.database) > 0 {
		_, _ = data.Write([]byte(c.database + "\x00"))
	}

	//认证插件名称
	if len(c.authPlugin) > 0 {
		_, _ = data.Write([]byte(c.authPlugin + "\x00"))
	}

	// logger.Debugf("客户端发送HandshakeResponse数据：% x", data.Bytes())

	return c.p.Write(data.Bytes())
}

func (c *Conn) Write(data []byte) error {
	c.p.Sequence = 0
	return c.p.Write(data)
}

func (c *Conn) Read() ([]byte, error) {
	return c.p.Read()
}

func (c *Conn) readOK() ([]byte, error) {
	data, err := c.p.Read()
	if err != nil {
		return data, err
	}

	if data[0] == OK_HEADER {
		return data, nil
	} else if data[0] == ERR_HEADER {
		return data, errors.New("exec cmd fail")
	} else {
		return data, errors.New("invalid ok packet")
	}
}

func (c *Conn) Close() error {
	_ = c.Quit()
	return c.p.Conn.Close()
}
