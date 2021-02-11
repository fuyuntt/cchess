package main

import (
	"bufio"
	"github.com/fuyuntt/cchess/ucci"
	"os"
)

func main() {
	engine := ucci.CreateEngine()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		ctx := ucci.CreateCmdCtx(os.Stdout)
		engine.ExecCommand(ctx, cmd)
	}
}
