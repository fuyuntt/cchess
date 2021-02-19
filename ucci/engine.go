package ucci

import (
	"fmt"
	"github.com/fuyuntt/cchess/ppos"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

type Engine struct {
	pos *ppos.Position
}

func (engine *Engine) ExecCommand(ctx *CmdCtx, cmdStr string) {
	logrus.Infof("cmd: %s", cmdStr)
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
	case "quit":
		engine.quit(ctx)
	}
}
func CreateEngine() *Engine {
	return &Engine{}
}
func (engine *Engine) ucci(ctx *CmdCtx) {
	ctx.fPrintln("id name FunChess 1.0")
	ctx.fPrintln("id copyright 2004-2006 www.fuyuntt.com")
	ctx.fPrintln("id author Fu Yun")
	ctx.fPrintln("id user 2004-2006 www.fuyuntt.com")

	ctx.fPrintln("option usemillisec type check")
	ctx.fPrintln("ucciok")
}

func (engine *Engine) isReady(ctx *CmdCtx) {
	ctx.fPrintln("readyok")
}

func (engine *Engine) position(positionStr string) {
	position, err := ppos.CreatePositionFromPosStr(positionStr)
	if err != nil {
		logrus.Errorf("parse position failure, position: %s, err: %v", positionStr, err)
	}
	engine.pos = position
}

func (engine *Engine) goThink(ctx *CmdCtx) {
	moves, vl := engine.pos.SearchMain(3 * time.Second)
	logrus.Infof("moves: %v, vl %d", moves, vl)
	ctx.fPrintln("bestmove " + moves[0].String())
}

func (engine *Engine) quit(ctx *CmdCtx) {
	ctx.fPrintln("bye")
}

type CmdCtx struct {
	output io.Writer
}

func CreateCmdCtx(writer io.Writer) *CmdCtx {
	return &CmdCtx{writer}
}

func (ctx *CmdCtx) fPrintln(a ...interface{}) {
	logrus.Infof("ucci: %v", a)
	_, err := fmt.Fprintln(ctx.output, a...)
	if err != nil {
		logrus.Errorf("output write failure. %v, err=%v", a, err)
	}
}
