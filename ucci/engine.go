package ucci

import (
	"fmt"
	"github.com/fuyuntt/cchess/cchess"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

type Engine struct {
	pos *cchess.Position
}

func (engine *Engine) ExecCommand(ctx *CmdCtx, cmdStr string) {
	cmdParam := strings.SplitN(cmdStr, " ", 2)
	switch cmdParam[0] {
	case "ucci":
		engine.ucci(ctx)
	case "isready":
		engine.isReady(ctx)
	case "position":
		engine.position(cmdParam[1])
	case "go":
		engine.goThink(ctx)
	}
}
func CreateEngine() *Engine {
	return &Engine{}
}
func (engine *Engine) ucci(ctx *CmdCtx) {
	ctx.fPrintln("id author Fu Yun")
	ctx.fPrintln("option usemillisec type check")
	ctx.fPrintln("ucci ok")
}

func (engine *Engine) isReady(ctx *CmdCtx) {
	ctx.fPrintln("readyok")
}

func (engine *Engine) position(fen string) {
	position, err := parsePosition(fen)
	if err != nil {
		logrus.Errorf("parse position failure, position: %s, err: %v", fen, err)
	}
	engine.pos = position
}

func (engine *Engine) goThink(ctx *CmdCtx) {
	move, vl := engine.pos.SearchMain(3 * time.Second)
	mvStr := convertMv(move)
	logrus.Infof("move: %s, vl %d", mvStr, vl)
	ctx.fPrintln("bestmove " + mvStr)
}

type CmdCtx struct {
	output io.Writer
}

func CreateCmdCtx(writer io.Writer) *CmdCtx {
	return &CmdCtx{writer}
}

func (ctx *CmdCtx) fPrintln(a ...interface{}) {
	_, err := fmt.Fprintln(ctx.output, a...)
	if err != nil {
		fmt.Println("output write failure. ", a)
	}
}

func convertMv(mv cchess.Move) string {
	return string([]rune{rune('a' + mv.Src().GetX() - 3), rune('0' + mv.Src().GetY() - 3), rune('a' + mv.Dst().GetX() - 3), rune('0' + mv.Dst().GetY() - 3)})
}
