package server

//用于接受连接

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync/atomic"

	"github.com/hsyan2008/sunder/config"
	. "github.com/hsyan2008/sunder/mysql"
)

var DEFAULT_CAPABILITY uint32 = CLIENT_LONG_PASSWORD | CLIENT_LONG_FLAG |
	CLIENT_CONNECT_WITH_DB | CLIENT_PROTOCOL_41 |
	CLIENT_TRANSACTIONS | CLIENT_SECURE_CONNECTION //|
// CLIENT_PLUGIN_AUTH //|
// CLIENT_INTERACTIVE

var baseConnId uint32 = 10000

type Conn struct {
	p        *Packet
	listener net.Listener

	connectionId uint32
	salt         []byte
	status       uint16

	capability  uint32
	collationId CollationId

	authPlugin string

	username string
	password string
	//注意use db的时候要修改，client/conn.go要同步
	database string
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{
		p:            NewPacket(conn),
		connectionId: atomic.AddUint32(&baseConnId, 1),
		salt:         RandomSalt(20),
		status:       SERVER_STATUS_AUTOCOMMIT,
	}
}

func (c *Conn) Close() error {
	return c.p.Conn.Close()
}

func (c *Conn) Handshake(accounts map[string]config.Account) (err error) {
	err = c.WriteHandshake()
	if err != nil {
		return
	}
	err = c.ReadHandshakeResponse(accounts)
	if err != nil {
		_ = c.writeError(err)
		return
	}

	_ = c.writeOK()

	return err
}

func (c *Conn) WriteHandshake() error {
	// logger.Debugf("server WriteHandshake")
	data := make([]byte, 0, 512)
	data = append(data, MinProtocolVersion)
	data = append(data, ServerVersion...)
	data = append(data, 0x00)
	data = append(data, byte(c.connectionId), byte(c.connectionId>>8), byte(c.connectionId>>16), byte(c.connectionId>>24))
	data = append(data, c.salt[:8]...)
	data = append(data, 0x00)
	data = append(data, byte(DEFAULT_CAPABILITY), byte(DEFAULT_CAPABILITY>>8))
	data = append(data, byte(DEFAULT_COLLATION_ID))
	data = append(data, byte(c.status), byte(c.status>>8))
	data = append(data, byte(DEFAULT_CAPABILITY>>16), byte(DEFAULT_CAPABILITY>>24))
	if DEFAULT_CAPABILITY&CLIENT_PLUGIN_AUTH > 0 {
		data = append(data, byte(len(AUTH_NAME)))
	} else {
		data = append(data, 0x00)
		// data = append(data, 0x15)
	}
	data = append(data, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	data = append(data, c.salt[8:]...)
	data = append(data, 0x00)
	if DEFAULT_CAPABILITY&CLIENT_PLUGIN_AUTH > 0 {
		data = append(data, AUTH_NAME...)
		data = append(data, 0x00)
	}

	// logger.Debugf("服务端发送Handshake数据：% x", data)

	return c.p.Write(data)
}

func (c *Conn) ReadHandshakeResponse(accounts map[string]config.Account) error {
	// logger.Debugf("server ReadHandshakeResponse")
	data, err := c.p.Read()
	if err != nil {
		return err
	}

	pos := 0

	c.capability = binary.LittleEndian.Uint32(data[:4])
	pos += 4

	//skip max packet length
	pos += 4

	c.collationId = CollationId(data[pos])
	pos += 1

	//skip reserved
	pos += 23

	c.username = string(data[pos : pos+bytes.IndexByte(data[pos:], 0x00)])
	if account, ok := accounts[c.username]; ok {
		c.password = account.Password
	}
	pos += len(c.username) + 1

	authLen := int(data[pos])
	pos++

	auth := data[pos : pos+authLen]
	checkAuth := Sha1Password(c.password, c.salt)
	if !bytes.Equal(auth, checkAuth) {
		return NewDefaultError(ER_ACCESS_DENIED_ERROR, c.p.Conn.RemoteAddr().String(), c.username, "Yes")
	}

	pos += authLen

	if c.capability&CLIENT_CONNECT_WITH_DB > 0 {
		c.database = string(data[pos : pos+bytes.IndexByte(data[pos:], 0x00)])
		pos += len(c.database) + 1
	}

	//密码验证应该在这步之后？ TODO
	if pos < len(data) && c.capability&CLIENT_PLUGIN_AUTH > 0 {
		c.authPlugin = string(data[pos : pos+bytes.IndexByte(data[pos:], 0x00)])
		pos += len(c.authPlugin) + 1
	}

	// logger.Debugf("服务端ReadHandshakeResponse % x", data)
	// logger.Debug(*c)
	return nil
}

func (c *Conn) writeOK() error {
	data := make([]byte, 0, 32)

	data = append(data, OK_HEADER)

	data = append(data, 0x00)
	data = append(data, 0x00)

	if c.capability&CLIENT_PROTOCOL_41 > 0 {
		data = append(data, byte(c.status), byte(c.status>>8))
		data = append(data, 0, 0)
	}

	return c.p.Write(data)
}

func (c *Conn) writeError(e error) error {
	var m *SqlError
	var ok bool
	if m, ok = e.(*SqlError); !ok {
		m = NewError(ER_UNKNOWN_ERROR, e.Error())
	}

	data := make([]byte, 0, 16+len(m.Message))

	data = append(data, ERR_HEADER)
	data = append(data, byte(m.Code), byte(m.Code>>8))

	if c.capability&CLIENT_PROTOCOL_41 > 0 {
		data = append(data, '#')
		data = append(data, m.State...)
	}

	data = append(data, m.Message...)

	return c.p.Write(data)
}

func (c *Conn) writeEOF(status uint16) error {
	data := make([]byte, 0, 9)

	data = append(data, EOF_HEADER)
	if c.capability&CLIENT_PROTOCOL_41 > 0 {
		data = append(data, 0, 0)
		data = append(data, byte(status), byte(status>>8))
	}

	return c.p.Write(data)
}
