package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fuyuntt/cchess/ucci"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
)

var serverMode = flag.Bool("s", false, "open server mode")
var port = flag.Int("p", 1234, "server mode listening port")

func main() {
	flag.Parse()
	if *serverMode {
		networkEngine(*port)
	} else {
		file, err := os.Create("chess.log")
		if err == nil {
			logrus.SetOutput(file)
		}
		deal(os.Stdin, os.Stdout)
	}
}

// 网络引擎 可配合客户端使用
func networkEngine(port int) {
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		logrus.Errorf("监听端口失败, err=%v", err)
		return
	}
	logrus.Infof("start listening: %v", listen.Addr())
	for {
		conn, err := listen.Accept()
		if err != nil {
			logrus.Errorf("获取连接失败， err=%v", err)
			return
		}
		logrus.Infof("accept connection: %v", conn.RemoteAddr())
		go deal(conn, conn)
	}
}

func deal(reader io.Reader, writer io.Writer) {
	engine := ucci.CreateEngine()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		cmd := scanner.Text()
		ctx := ucci.CreateCmdCtx(writer)
		engine.ExecCommand(ctx, cmd)
		if cmd == "quit" {
			logrus.Infof("engine quit")
			return
		}
	}
}
