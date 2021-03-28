package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fuyuntt/cchess/client"
	"github.com/fuyuntt/cchess/ucci"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var serverMode = flag.Bool("s", false, "open server mode")
var port = flag.Int("p", 1234, "server mode listening port")

type MyFormatter struct{}

func (s *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("2006/01/02 15:04:05")
	msg := fmt.Sprintf("%s [%s] %s\n", timestamp, strings.ToUpper(entry.Level.String()), entry.Message)
	return []byte(msg), nil
}

func main() {
	flag.Parse()
	file, err := os.Create("chess.log")
	if err == nil {
		logrus.SetOutput(file)
		logrus.SetFormatter(&MyFormatter{})
	}
	if *serverMode {
		networkEngine(*port)
	} else {
		deal(os.Stdin, os.Stdout)
	}
}

// 网络引擎 需配合gui客户端使用
func networkEngine(port int) {
	http.HandleFunc("/api/is-legal-move", client.LegalMove)
	http.HandleFunc("/api/think", client.Think)
	logrus.Infof("start http server on port: %d", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	logrus.Errorf("stop server. err=%v", err)
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
