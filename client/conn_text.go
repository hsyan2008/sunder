package client

import (
	"time"

	"github.com/hsyan2008/go-logger/logger"
	. "github.com/hsyan2008/sunder/mysql"
)

func (c *Conn) Ping() (err error) {
	// err = c.p.Write([]byte{0x10}) //mariadb
	// err = c.p.Write([]byte{14}) //mysql
	err = c.p.Write([]byte{COM_PING}) //mysql
	if err != nil {
		// logger.Warn("send ping fail")
		return
	}
	_, err = c.readOK()
	// logger.Info("ping result", err)

	c.lastPingTime = time.Now().Unix()
	c.p.Sequence = 0

	return
}

func (c *Conn) Quit() (err error) {
	err = c.p.Write([]byte{COM_QUIT})
	if err != nil {
		logger.Warn("send quit fail")
		return
	}
	//quit不一定有返回，可能直接关闭连接
	// err = c.readOK()
	// logger.Info("quit result", err)

	c.p.Sequence = 0

	return
}

func (c *Conn) InitDb(name string) (data []byte, err error) {
	err = c.p.Write(append([]byte{COM_INIT_DB}, name+"\x00"...))
	if err != nil {
		// logger.Warn("send InitDb", name, "fail")
		return
	}
	data, err = c.readOK()
	// logger.Info("InitDb", name, "result", err)

	if err == nil {
		c.database = name
	}

	c.lastPingTime = time.Now().Unix()
	c.p.Sequence = 0

	return
}

//mariadb10.2后才支持，mysql不确定，5.6不支持
func (c *Conn) Reset() (data []byte, err error) {
	err = c.p.Write([]byte{COM_RESET_CONNECTION})
	if err != nil {
		// logger.Warn("send COM_RESET_CONNECTION fail")
		return
	}
	data, err = c.readOK()
	// logger.Info("COM_RESET_CONNECTION result", err)

	c.lastPingTime = time.Now().Unix()
	c.p.Sequence = 0

	return
}
