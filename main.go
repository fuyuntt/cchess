package main

import (
	"bufio"
	"github.com/fuyuntt/cchess/ucci"
	"github.com/sirupsen/logrus"
	"io"
	"net"
)

func main() {
	listen, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		logrus.Errorf("监听端口失败, err=%v", err)
		return
	}
	logrus.Infof("start listening: %v", listen.Addr())
	for {
		accept, err := listen.Accept()
		if err != nil {
			logrus.Errorf("获取连接失败， err=%v", err)
			return
		}
		logrus.Infof("accept connection: %v", accept.RemoteAddr())
		go deal(accept)
	}
}

func deal(readWriter io.ReadWriter) {
	engine := ucci.CreateEngine()
	scanner := bufio.NewScanner(readWriter)
	for scanner.Scan() {
		cmd := scanner.Text()
		ctx := ucci.CreateCmdCtx(readWriter)
		engine.ExecCommand(ctx, cmd)
		if cmd == "quit" {
			logrus.Infof("engine quit")
			return
		}
	}
}
