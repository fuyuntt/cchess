package main

import (
	"bufio"
	"flag"
	"github.com/fuyuntt/cchess/client"
	"github.com/fuyuntt/cchess/ucci"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
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

// 网络引擎 需配合gui客户端使用
func networkEngine(port int) {
	http.HandleFunc("/is-legal-move", client.LegalMove)
	http.HandleFunc("/think", client.Think)
	_ = http.ListenAndServe(":"+strconv.Itoa(port), nil)
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
