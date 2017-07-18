package server

import (
	"sync/atomic"
	"time"

	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/sunder/client"
	"github.com/hsyan2008/sunder/config"
)

func (s *Server) createWrCon(kind client.Kind) {
	var host config.Host
	var wr *writeReader
	if kind == client.WRITER {
		wr = s.writer
		host = s.cfg.Write
	} else {
		wr = s.reader
		host = s.cfg.Reads[s.seq%uint64(len(s.cfg.Reads))]
		atomic.AddUint64(&(s.seq), 1)
	}
	// logger.Info(host)
	if host.Addr == "" || host.Username == "" || host.Password == "" {
		logger.Error("invalid host config", host)
		panic("invalid host config")
	}
	con, err := client.NewConn(host.Addr, host.Username, host.Password, "")
	if err != nil {
		logger.Warn("client dial fail", err)
		panic("client dial fail")
	}
	err = con.Handshake()
	if err != nil {
		_ = con.Close()
		logger.Warn("client Handshake fail", err)
		panic("client Handshake fail")
	}

	con.Kind = wr.kind
	wr.channel <- con
	atomic.AddUint32(&(wr.count), 1)
	atomic.AddUint32(&(wr.idleCount), 1)
	// logger.Warnf("create kind:%d count:%d idleCount:%d", wr.kind, wr.count, wr.idleCount)
}

func (s *Server) getWrCon(kind client.Kind) (c *client.Conn) {
	var wr *writeReader
	if kind == client.READER {
		//表示没有初始化过，即没有配置
		if s.reader.count == 0 {
			return nil
		}
		wr = s.reader
	} else {
		wr = s.writer
	}

	select {
	case c = <-wr.channel:
	default:
		s.createWrCon(kind)
		c = <-wr.channel
	}

	atomic.AddUint32(&(wr.idleCount), ^uint32(0))

	go s.completeCon(kind, s.cfg.MaxIdle-wr.idleCount)

	return
}

//补充连接数量达到MaxIdle限制
func (s *Server) completeCon(kind client.Kind, count uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for count > 0 {
		s.createWrCon(kind)
		count--
	}
}

func (s *Server) ping(wr *writeReader) {
	for c := range wr.channel {
		len := s.cfg.KeepAlive - (time.Now().Unix() - c.GetLastPingTime())
		go func(c *client.Conn) {
			err := c.Ping()
			if err != nil {
				atomic.AddUint32(&(wr.count), ^uint32(0))
				atomic.AddUint32(&(wr.idleCount), ^uint32(0))
				_ = c.Close()
			} else {
				wr.channel <- c
			}
		}(c)
		//如果最近检测的连接还没到时间，就停止一下
		if len > 0 {
			time.Sleep(time.Second * time.Duration(len))
		}
	}
}
