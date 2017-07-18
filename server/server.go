package server

//用于接受连接

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/sunder/client"
	"github.com/hsyan2008/sunder/config"
	"github.com/hsyan2008/sunder/hack"
	. "github.com/hsyan2008/sunder/mysql"
	"github.com/hsyan2008/sunder/sqlparser"
)

type Server struct {
	cfg      config.Instance
	listener net.Listener

	running bool

	writer *writeReader
	reader *writeReader

	//轮流读取从库的配置
	seq uint64

	mu *sync.Mutex
}

type writeReader struct {
	kind      client.Kind
	channel   chan *client.Conn
	count     uint32
	idleCount uint32
}

func NewServer(instance config.Instance) (*Server, error) {

	s := new(Server)
	s.mu = new(sync.Mutex)
	s.writer = &writeReader{kind: client.WRITER}
	s.reader = &writeReader{kind: client.READER}
	s.cfg = instance
	var err error

	if !instance.Enabled || instance.Bind == "" {
		return nil, errors.New("invalid config")
	}

	// test server
	s.listener, err = net.Listen("tcp", instance.Bind)
	if err != nil {
		logger.Error("listen ", instance.Bind, " error ", err)
	}

	//创建client队列
	s.writer.channel = make(chan *client.Conn, instance.MaxIdle)
	var i uint32
	for i = 0; i < instance.MaxIdle; i++ {
		go s.createWrCon(client.WRITER)
	}
	s.reader.channel = make(chan *client.Conn, instance.MaxIdle)
	for i = 0; i < instance.MaxIdle; i++ {
		go s.createWrCon(client.READER)
	}

	go s.ping(s.writer)
	go s.ping(s.reader)

	return s, err
}

func (s *Server) Close() {
	defer logger.Warn("server close")
	s.running = false
	if s.listener != nil {
		// close(s.writer.channel)
		// close(s.reader.channel)
		_ = s.listener.Close()
	}
}

func (s *Server) Run() {
	s.running = true

	for s.running {
		con, err := s.listener.Accept()

		if err != nil {
			logger.Warn("accept error ", err)
			continue
		}

		go s.dispatch(con)
	}
}

func (s *Server) dispatch(con net.Conn) {
	var err error
	//recover
	defer func() {
		if err := recover(); err != nil {
			logger.Warn("recover:", err)
		}
	}()

	serCon := NewConn(con)
	defer func() {
		_ = serCon.Close()
	}()
	err = serCon.Handshake(s.cfg.Accounts)
	if err != nil {
		logger.Warn("server Handshake fail", err)
		return
	}

	c, err := s.getClientConn(client.READER, serCon.database)
	if err != nil {
		logger.Warn("getClientConn fail", err)
		return
	}
	// defer s.putClientConn(c)
	defer func() {
		_ = c.Close()
	}()

	go copyClient(serCon, c)

	var stmt sqlparser.Statement
	for {
		logger.Debug("ready to read")
		serCon.p.Sequence = 0

		rs, err := serCon.p.Read()
		// logger.Warn(rs, err)
		if err != nil {
			logger.Warn(err)
			return
		}

		cmd := rs[0]
		data := rs[1:]
		logger.Warn(cmd, string(data))

		stmt, err = sqlparser.Parse(string(data))
		logger.Warn(stmt, err)
		logger.Warnf("%#v", stmt)
		// switch v := stmt.(type) {
		// case *sqlparser.Select:
		// 	logger.Warnf("%#v", v.From)
		// 	logger.Warnf("%#v", v.From[0])
		// 	logger.Warnf("%#v", v.From[0].(*sqlparser.AliasedTableExpr).Expr)
		// 	tableName := v.From[0].(*sqlparser.AliasedTableExpr).Expr.(*sqlparser.TableName)
		// 	logger.Warnf("%#v", tableName.Name)
		// 	logger.Warn(string(tableName.Name))
		// }

		switch cmd {
		case COM_QUIT:
			return
		case COM_INIT_DB:
			name := hack.String(data)
			serCon.database = name
			_ = c.Write(rs)
		case COM_QUERY: //php调用pdo，都是这个

			if c.Kind == client.READER && IsWrite(string(data)) {
				_ = c.Close()
				c, err = s.getClientConn(client.WRITER, serCon.database)
				if err != nil {
					logger.Warn("getClientConn fail", err)
					return
				}
				go copyClient(serCon, c)
			}
			_ = c.Write(rs)
		case COM_FIELD_LIST:
			_ = c.Write(rs)
		case COM_PING:
			_ = serCon.writeOK()
		case COM_STMT_PREPARE: //golang通过xorm，都是这个
			_ = c.Write(rs)
		case COM_STMT_EXECUTE:
			_ = c.Write(rs)
		// case COM_STMT_FETCH:
		// 	_ = c.Write(rs)
		case COM_STMT_CLOSE:
			_ = c.Write(rs)
		// case COM_STMT_SEND_LONG_DATA:
		// 	_ = c.Write(rs)
		// case COM_STMT_RESET:
		// 	_ = c.Write(rs)
		default:
			msg := fmt.Sprintf("command %d not supported now", cmd)
			logger.Warn(msg)
			_ = serCon.writeError(NewError(ER_UNKNOWN_ERROR, msg))
		}

	}
}

//如何中断？ TODO 暂时直接断开连接
func copyClient(sc *Conn, cc *client.Conn) {
	_, _ = io.Copy(sc.p.Conn, cc.GetConn())
}

func (s *Server) getClientConn(kind client.Kind, database string) (c *client.Conn, err error) {
	if kind == client.READER {
		logger.Info("get reader conn")
		c = s.getWrCon(client.READER)
	}
	if c == nil {
		logger.Info("get writer conn")
		c = s.getWrCon(client.WRITER)
	}

	if c != nil && database != "" {
		_, err = c.InitDb(database)
		if err != nil {
			logger.Warn("initdb ", err)
			// _ = serCon.writeError(NewDefaultError(ER_NO_DB_ERROR))
			return nil, err
		}
	}

	return c, nil
}

func (s *Server) putClientConn(c *client.Conn) {
	if c.Kind == client.READER {
		logger.Info("reuse reader")
		atomic.AddUint32(&(s.reader.idleCount), 1)
		s.reader.channel <- c
	} else {
		logger.Info("reuse writer")
		atomic.AddUint32(&(s.writer.idleCount), 1)
		s.writer.channel <- c
	}

}
