package client

import (
	"encoding/json"
	"github.com/fuyuntt/cchess/ppos"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func LegalMove(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	query := req.URL.Query()
	position := query.Get("position")
	mv := query.Get("move")
	pos, err := ppos.CreatePositionFromPosStr(position)
	if err != nil {
		logrus.Errorf("create position failure. err=%v", err)
		return
	}
	legal := pos.LegalMove(ppos.GetMoveFromICCS(mv))
	marshal, _ := json.Marshal(map[string]interface{}{"isLegal": legal})
	_, _ = resp.Write(marshal)
}

func Think(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)
	query := req.URL.Query()
	position := query.Get("position")
	pos, err := ppos.CreatePositionFromPosStr(position)
	if err != nil {
		logrus.Errorf("create position failure. err=%v", err)
		return
	}
	res, score := pos.SearchMain(3 * time.Second)
	var moves []string
	for _, mv := range res {
		moves = append(moves, mv.ICCS())
	}
	logrus.Infof("think result, score: %d, moves:%v", score, moves)
	marshal, _ := json.Marshal(map[string]interface{}{"moves": moves, "score": score})
	_, _ = resp.Write(marshal)
}
