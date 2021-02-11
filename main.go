package main

import (
	"bufio"
	"github.com/fuyuntt/cchess/ucci"
	"os"
)

func main() {
	engine := ucci.CreateEngine()
	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		ctx := ucci.CreateCmdCtx(os.Stdout)
		engine.ExecCommand(ctx, cmd[:len(cmd)-1])
	}
}
