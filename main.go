package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/sunder/config"
	"github.com/hsyan2008/sunder/server"
)

func main() {

	var cfg config.Config
	_, err := toml.DecodeFile("config.toml", &cfg)
	if err != nil {
		logger.Error("config load fail", err)
		os.Exit(1)
	}
	// logger.Debugf("%#v", cfg)

	setLog(cfg.Log)

	serverChannel := make(chan *server.Server, len(cfg.Instances))
	for _, val := range cfg.Instances {
		s, err := server.NewServer(val)
		if err != nil {
			logger.Error("create server fail:", err)
			continue
		}

		serverChannel <- s
		go s.Run()
	}
	close(serverChannel)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	sig := <-sc
	logger.Infof("Got signal [%d] to exit.", sig)
	for s := range serverChannel {
		s.Close()
	}
}

//初始化log写入文件
func setLog(lc config.Log) {
	logger.SetLevelStr(lc.Level)
	logger.SetConsole(lc.Console)
	logger.SetLogGoID(lc.Goid)

	if lc.File != "" {
		if lc.Type == "daily" {
			logger.SetRollingDaily(lc.File)
		} else if lc.Type == "roll" {
			logger.SetRollingFile(lc.File, lc.Maxnum, lc.Size, lc.Unit)
		} else {
			logger.Warn("请设置log存储方式")
		}
	} else {
		logger.Warn("没有设置log目录和文件")
	}
}
