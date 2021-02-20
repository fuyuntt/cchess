package ppos

import (
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
	"time"
)

// 最大深度
const limitDepth = 64

// 初始move长度
const initMovesSize = 40

// 杀棋分
const mateValue = 10000

// 和棋分
const drawValue = -20

// 搜索出胜局的分数
const winValue = mateValue - 100

// 先行优势
const advancedValue = 3

type searchCtx struct {
	// 已搜索的局面数
	nPositionCount int
	// 停止搜索的时间
	stopSearchTime time.Time
	// 停止搜索
	stopSearch bool
	// 开始搜索时局面的distance
	initDistance int
	// 此次搜索最大距离
	maxDistance int
	// 历史表
	historyMoveTable [65536]int
	// hash表
	historyPosTable [65536]historyPosition
}

const hashIdxMask = 0xffff

func (ctx *searchCtx) probeHash(zob ZobristHash, depth int, alpha int, beta int) (bool, int) {
	hisPosition := &ctx.historyPosTable[zob&hashIdxMask]
	if hisPosition.zobrist != zob || hisPosition.depth < depth {
		return false, 0
	}
	if hisPosition.posType == hisExact {
		return true, hisPosition.value
	} else if hisPosition.posType == hisAlpha && hisPosition.value <= alpha {
		return true, alpha
	} else if hisPosition.posType == hisBeta && hisPosition.value >= beta {
		return true, beta
	}
	return false, 0
}
func (ctx *searchCtx) recordHash(zob ZobristHash, depth int, value int, posType historyType) {
	hisPosition := &ctx.historyPosTable[zob&hashIdxMask]
	if hisPosition.zobrist == zob && hisPosition.depth > depth {
		return
	}
	hisPosition.zobrist = zob
	hisPosition.depth = depth
	hisPosition.posType = posType
	hisPosition.value = value
}

type historyMove struct {
	move       Move
	pcCaptured Piece
	checked    bool
	posZobrist ZobristHash
}
type historyType uint8

const (
	hisExact historyType = iota
	hisAlpha
	hisBeta
)

type historyPosition struct {
	zobrist ZobristHash
	depth   int
	posType historyType
	value   int
}
type Position struct {
	// 棋盘
	pcSquares [256]Piece
	// 该哪方走
	playerSd Side
	// 红棋分
	vlRed int
	// 黑旗分
	vlBlack int
	// 局面zobrist
	zobrist ZobristHash
	// 走棋栈，可从中找到是否有重复局面
	mvStack []historyMove
	// 距离根节点的步数
	nDistance int
}

func (pos *Position) ChangeSide() {
	pos.playerSd = pos.playerSd.OpSide()
	pos.zobrist ^= playerZobrist
}
func (pos *Position) AddPiece(sq Square, pc Piece) {
	pos.pcSquares[sq] = pc
	if pc == PcNop {
		return
	}
	side := pc.GetSide()
	pcValue := pieceValue[pc.GetType()]
	if side == SdRed {
		pos.vlRed += pcValue[sq]
	} else {
		pos.vlBlack += pcValue[sq.Flip()]
	}
	pos.zobrist ^= GetZobrist(sq, pc)
}
func (pos *Position) DelPiece(sq Square) Piece {
	pcCaptured := pos.pcSquares[sq]
	if pcCaptured == PcNop {
		return PcNop
	}
	pos.pcSquares[sq] = PcNop
	side := pcCaptured.GetSide()
	pcValueTable := pieceValue[pcCaptured.GetType()]
	if side == SdRed {
		pos.vlRed -= pcValueTable[sq]
	} else {
		pos.vlBlack -= pcValueTable[sq.Flip()]
	}
	pos.zobrist ^= GetZobrist(sq, pcCaptured)
	return pcCaptured
}

func (pos *Position) Evaluate() int {
	if pos.playerSd == SdRed {
		return pos.vlRed - pos.vlBlack + advancedValue
	} else {
		return pos.vlBlack - pos.vlRed + advancedValue
	}
}
func (pos *Position) MovePiece(mv Move) Piece {
	var sqSrc, sqDst = mv.Src(), mv.Dst()
	var pcSrc, pcDst = pos.DelPiece(sqSrc), pos.DelPiece(sqDst)
	pos.AddPiece(sqDst, pcSrc)
	return pcDst
}
func (pos *Position) UndoMovePiece(mv Move, pcCaptured Piece) {
	var sqSrc, sqDst = mv.Src(), mv.Dst()
	var _, pcDst = pos.DelPiece(sqSrc), pos.DelPiece(sqDst)
	pos.AddPiece(sqSrc, pcDst)
	pos.AddPiece(sqDst, pcCaptured)
}
func (pos *Position) Checked() bool {
	sqSrc := SqStart
	selfKing := GetPiece(PtKing, pos.playerSd)
	for sqSrc = SqStart; sqSrc <= SqEnd && pos.pcSquares[sqSrc] != selfKing; sqSrc++ {
	}
	if sqSrc > SqEnd {
		return false
	}
	opSide := pos.playerSd.OpSide()
	// 1. 判断是否被对方的兵(卒)将军
	if pos.pcSquares[sqForward(sqSrc, pos.playerSd)] == GetPiece(PtPawn, opSide) {
		return true
	}
	for delta := Square(-0x01); delta <= 0x01; delta += 0x02 {
		if pos.pcSquares[sqSrc+delta] == GetPiece(PtPawn, opSide) {
			return true
		}
	}

	// 2. 判断是否被对方的马将军
	opKnight := GetPiece(PtKnight, opSide)
	for i := 0; i < 8; i++ {
		sqDst := sqSrc + knightMoveTab[i]
		if pos.pcSquares[sqDst] == opKnight && pos.pcSquares[getKnightPin(sqDst, sqSrc)] == PcNop {
			return true
		}
	}
	// 3. 判断是否被对方的车或炮将军(包括将帅对脸)
	opRook := GetPiece(PtRook, opSide)
	opCannon := GetPiece(PtCannon, opSide)
	opKing := GetPiece(PtKing, opSide)
	for i := 0; i < 4; i++ {
		delta := lineMoveDelta[i]
		sqDst := sqSrc + delta
		for ; sqDst.InBoard() && pos.pcSquares[sqDst] == PcNop; sqDst += delta {
		}
		if !sqDst.InBoard() {
			continue
		}
		pcDst := pos.pcSquares[sqDst]
		if pcDst == opRook || pcDst == opKing {
			return true
		}
		sqDst += delta
		for ; sqDst.InBoard() && pos.pcSquares[sqDst] == PcNop; sqDst += delta {
		}
		if !sqDst.InBoard() {
			continue
		}
		if pos.pcSquares[sqDst] == opCannon {
			return true
		}
	}
	return false
}

// onlyCapture=ture  只生成吃子的走法，否则生成所有走法
func (pos *Position) GenerateMoves(moves []Move, onlyCapture bool) []Move {
	var testCapture = func(pcDst Piece) bool {
		return !onlyCapture || pcDst != PcNop
	}
	for sqSrc := SqStart; sqSrc <= SqEnd; sqSrc++ {
		pcSrc := pos.pcSquares[sqSrc]
		if pcSrc.GetSide() != pos.playerSd {
			continue
		}
		switch pcSrc.GetType() {
		case PtKing:
			for i := 0; i < 4; i++ {
				var sqDst = sqSrc + lineMoveDelta[i]
				if !sqDst.InFort() {
					continue
				}
				pcDst := pos.pcSquares[sqDst]
				if pcDst.GetSide() != pos.playerSd && testCapture(pcDst) {
					moves = append(moves, GetMove(sqSrc, sqDst))
				}
			}
		case PtAdvisor:
			for i := 0; i < 4; i++ {
				sqDst := sqSrc + advisorMoveTab[i]
				if !sqDst.InFort() {
					continue
				}
				pcDst := pos.pcSquares[sqDst]
				if pcDst.GetSide() != pos.playerSd && testCapture(pcDst) {
					moves = append(moves, GetMove(sqSrc, sqDst))
				}
			}
		case PtBishop:
			for i := 0; i < 4; i++ {
				sqDst := sqSrc + bishopMoveTab[i]
				if !sqDst.InBoard() || sqDst.GetSide() != pos.playerSd {
					continue
				}
				sqBishopPin := (sqSrc + sqDst) >> 1
				if pos.pcSquares[sqBishopPin] != PcNop {
					continue
				}
				pcDst := pos.pcSquares[sqDst]
				if pcDst.GetSide() != pos.playerSd && testCapture(pcDst) {
					moves = append(moves, GetMove(sqSrc, sqDst))
				}
			}
		case PtKnight:
			for i := 0; i < 8; i++ {
				sqDst := sqSrc + knightMoveTab[i]
				if !sqDst.InBoard() {
					continue
				}
				sqPin := getKnightPin(sqSrc, sqDst)
				if pos.pcSquares[sqPin] != PcNop {
					continue
				}
				pcDst := pos.pcSquares[sqDst]
				if pcDst.GetSide() != pos.playerSd && testCapture(pcDst) {
					moves = append(moves, GetMove(sqSrc, sqDst))
				}
			}
		case PtRook:
			for i := 0; i < 4; i++ {
				sqDelta := lineMoveDelta[i]
				for sqDst := sqSrc + sqDelta; sqDst.InBoard(); sqDst += sqDelta {
					pcDst := pos.pcSquares[sqDst]
					if pcDst == PcNop {
						if !onlyCapture {
							moves = append(moves, GetMove(sqSrc, sqDst))
						}
						continue
					} else {
						if pcDst.GetSide() != pos.playerSd {
							moves = append(moves, GetMove(sqSrc, sqDst))
						}
						break
					}
				}
			}
		case PtCannon:
			for i := 0; i < 4; i++ {
				sqDelta := lineMoveDelta[i]
				sqDst := sqSrc + sqDelta
				for ; sqDst.InBoard(); sqDst += sqDelta {
					pcDst := pos.pcSquares[sqDst]
					if pcDst == PcNop {
						if !onlyCapture {
							moves = append(moves, GetMove(sqSrc, sqDst))
						}
						continue
					} else {
						break
					}
				}
				sqDst += sqDelta
				for ; sqDst.InBoard(); sqDst += sqDelta {
					pcDst := pos.pcSquares[sqDst]
					if pcDst != PcNop {
						if pcDst.GetSide() != pos.playerSd {
							moves = append(moves, GetMove(sqSrc, sqDst))
						}
						break
					}
				}
			}
		case PtPawn:
			sqDst := sqForward(sqSrc, pos.playerSd)
			pcDst := pos.pcSquares[sqDst]
			if sqDst.InBoard() && pcDst.GetSide() != pos.playerSd && testCapture(pcDst) {
				moves = append(moves, GetMove(sqSrc, sqDst))
			}
			if sqSrc.GetSide() != pos.playerSd {
				for delta := Square(-0x01); delta <= 0x01; delta += 0x02 {
					sqDst := sqSrc + delta
					pcDst := pos.pcSquares[sqDst]
					if sqDst.InBoard() && pcDst.GetSide() != pos.playerSd && testCapture(pcDst) {
						moves = append(moves, GetMove(sqSrc, sqDst))
					}
				}
			}
		}
	}
	return moves
}

// 走棋 会变更当前走棋方
func (pos *Position) MakeMove(move Move) bool {
	preZob := pos.zobrist
	pcCaptured := pos.MovePiece(move)
	if move != MvNop && pos.Checked() {
		pos.UndoMovePiece(move, pcCaptured)
		return false
	}
	pos.ChangeSide()
	pos.mvStack = append(pos.mvStack, historyMove{move, pcCaptured, pos.Checked(), preZob})
	pos.nDistance++
	return true
}

func (pos *Position) UndoMakeMove() {
	pos.ChangeSide()
	moveHis := pos.mvStack[pos.nDistance]
	pos.UndoMovePiece(moveHis.move, moveHis.pcCaptured)
	pos.nDistance--
	pos.mvStack = pos.mvStack[:pos.nDistance+1]
}

// return 评价值，主要变例(逆序）
func (pos *Position) searchAlphaBeta(ctx *searchCtx, vlAlpha, vlBeta, depth int) (int, []Move) {
	tickSearch(ctx)
	success, vl := ctx.probeHash(pos.zobrist, depth, vlAlpha, vlBeta)
	if success {
		return vl, nil
	}
	if pos.nDistance > ctx.initDistance {
		// 1. 到达水平线，使用静态局面搜索
		if depth <= 0 {
			return pos.searchQuiescent(ctx, vlAlpha, vlBeta)
		}

		// 1-1. 检查重复局面
		rep, vl := pos.CheckReputation()
		if rep {
			return vl, nil
		}

		// 1-2. 到达极限深度就返回局面评价
		if pos.nDistance == ctx.maxDistance {
			return pos.Evaluate(), nil
		}
	}

	moves := make([]Move, 0, initMovesSize)
	vlBest := -mateValue
	var pvMovesBest []Move
	var mvBest = MvNop
	moves = pos.GenerateMoves(moves, false)
	sort.Sort(MoveSorter{moves: moves, eval: func(mv Move) int {
		return ctx.historyMoveTable[mv]
	}})
	for _, mv := range moves {
		if !pos.MakeMove(mv) {
			continue
		}
		vl, pvMoves := pos.searchAlphaBeta(ctx, -vlBeta, -vlAlpha, depth-1)
		vl = -vl
		pos.UndoMakeMove()
		if ctx.stopSearch {
			return 0, nil
		}
		if vl > vlBest {
			vlBest = vl
			if vl >= vlBeta {
				mvBest = mv
				break
			}
			if vl > vlAlpha {
				mvBest = mv
				vlAlpha = vl
				pvMovesBest = append(pvMoves, mv)
			}
		}
	}
	// 所有的move都无法走 杀棋!
	if vlBest == -mateValue {
		return pos.nDistance - ctx.initDistance - mateValue, nil
	}
	if mvBest != MvNop {
		ctx.historyMoveTable[mvBest] += depth * depth
	}
	if vlBest != vlAlpha {
		if vlBest >= vlBeta {
			ctx.recordHash(pos.zobrist, depth, vlBest, hisBeta)
		} else {
			ctx.recordHash(pos.zobrist, depth, vlBest, hisAlpha)
		}
		return vlBest, nil
	} else {
		ctx.recordHash(pos.zobrist, depth, vlBest, hisExact)
		return vlBest, pvMovesBest
	}
}

// 搜索中的状态检查
func tickSearch(searchCtx *searchCtx) {
	searchCtx.nPositionCount++
	if searchCtx.nPositionCount&0x1fff == 0 && time.Now().After(searchCtx.stopSearchTime) {
		searchCtx.stopSearch = true
	}
}
func (pos *Position) searchQuiescent(ctx *searchCtx, vlAlpha, vlBeta int) (int, []Move) {
	// 1. 检查重复局面
	rep, vl := pos.CheckReputation()
	if rep {
		return vl, nil
	}

	// 2. 到达极限深度就返回局面评价
	if pos.nDistance == ctx.maxDistance {
		return pos.Evaluate(), nil
	}

	// 3. 初始化最佳值
	vlBest := -mateValue
	var pvMoveBest []Move

	moves := make([]Move, 0, initMovesSize)
	if pos.InCheck() {
		// 4. 如果被将军，则生成全部走法
		moves = pos.GenerateMoves(moves, false)
		sort.Sort(MoveSorter{moves: moves, eval: func(mv Move) int {
			return ctx.historyMoveTable[mv]
		}})
	} else {
		// 5. 如果不被将军，先做局面评价
		vl = pos.Evaluate()
		if vl > vlBest {
			vlBest = vl
			if vl >= vlBeta {
				return vl, nil
			}
			if vl > vlAlpha {
				vlAlpha = vl
			}
		}

		// 6. 如果局面评价没有截断，再生成吃子走法
		moves = pos.GenerateMoves(moves, true)
		sort.Sort(MoveSorter{moves: moves, eval: func(mv Move) int {
			return pos.mvvLvaValue(mv)
		}})
	}

	// 7. 逐一走这些走法，并进行递归
	for _, mv := range moves {
		if !pos.MakeMove(mv) {
			continue
		}
		vl, pvMove := pos.searchQuiescent(ctx, -vlBeta, -vlAlpha)
		vl = -vl
		pos.UndoMakeMove()

		if vl > vlBest {
			vlBest = vl
			if vl >= vlBeta {
				return vl, nil
			}
			if vl > vlAlpha {
				vlAlpha = vl
				pvMoveBest = append(pvMove, mv)
			}
		}
	}

	if vlBest == -mateValue {
		return pos.nDistance - ctx.initDistance - mateValue, nil
	} else if vlBest != vlAlpha {
		return vlBest, nil
	} else {
		return vlBest, pvMoveBest
	}
}

func (pos *Position) mvvLvaValue(mv Move) int {
	return mvvLvaPieceValue[pos.pcSquares[mv.Dst()]]<<3 - mvvLvaPieceValue[pos.pcSquares[mv.Src()]]
}

func (pos *Position) InCheck() bool {
	if pos.nDistance == 0 {
		return pos.Checked()
	} else {
		return pos.mvStack[pos.nDistance].checked
	}
}

// 检查重复局面
// return 是否有重复局面， 重复局面的评分（输，赢，和）
func (pos *Position) CheckReputation() (bool, int) {
	selfSide := false
	selfAlwaysCheck, opAlwaysCheck := true, true
	for mvIdx := pos.nDistance; mvIdx > 0; mvIdx-- {
		moveHistory := pos.mvStack[mvIdx]
		// 吃子着法肯定不会重复
		if moveHistory.pcCaptured != PcNop {
			break
		}
		if selfSide {
			selfAlwaysCheck = selfAlwaysCheck && moveHistory.checked
			if moveHistory.posZobrist == pos.zobrist {
				return true, reputationValue(selfAlwaysCheck, opAlwaysCheck)
			}
		} else {
			opAlwaysCheck = opAlwaysCheck && moveHistory.checked
		}
		selfSide = !selfSide
	}
	return false, 0
}

func reputationValue(selfAlwaysCheck, opAlwaysCheck bool) int {
	vl := 0
	if selfAlwaysCheck {
		vl += -mateValue
	}
	if opAlwaysCheck {
		vl += mateValue
	}
	if vl == 0 {
		vl = -drawValue
	}
	return vl
}
func (pos *Position) SearchMain(duration time.Duration) ([]Move, int) {
	startTime := time.Now()
	effectiveEndTime := time.Now()
	ctx := &searchCtx{}
	ctx.stopSearchTime = time.Now().Add(duration)
	ctx.initDistance = pos.nDistance
	ctx.maxDistance = pos.nDistance + limitDepth
	var resValue int
	var resPvMove []Move
	nPositions := 0
	maxDepth := 0
	for ; maxDepth < limitDepth; maxDepth++ {
		ctx.nPositionCount = 0
		value, pvMoves := pos.searchAlphaBeta(ctx, -mateValue, mateValue, maxDepth)
		if ctx.stopSearch {
			maxDepth--
			break
		}
		resValue = value
		resPvMove = pvMoves
		nPositions = ctx.nPositionCount
		effectiveEndTime = time.Now()
		if resValue > winValue || resValue < -winValue {
			break
		}
	}
	revertSlice(resPvMove)
	logrus.Infof("search depth: %d, search nodes: %d, search time: %v, effect time:%v, score:%v, pv moves: %v", maxDepth, nPositions, time.Now().Sub(startTime), effectiveEndTime.Sub(startTime), resValue, resPvMove)
	return resPvMove, resValue
}
func (pos *Position) String() string {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteRune(rune('0' + 9 - i))
		sb.WriteRune(' ')
		sb.WriteString(pos.pcSquares[GetSquare(0, i)].String())
		for j := 1; j < 9; j++ {
			sb.WriteRune('-')
			sb.WriteString(pos.pcSquares[GetSquare(j, i)].String())
		}
		sb.WriteRune('\n')
	}
	sb.WriteString("  a")
	for i := 1; i < 9; i++ {
		sb.WriteRune(' ')
		sb.WriteRune(rune('a' + i))
	}
	sb.WriteRune('\n')
	return sb.String()
}
func (pos *Position) FenString() string {
	var sb strings.Builder
	for y := 0; y < 10; y++ {
		var spaceCount int
		for x := 0; x < 9; x++ {
			pc := pos.pcSquares[GetSquare(x, y)]
			if pc == PcNop {
				spaceCount++
			} else {
				if spaceCount != 0 {
					sb.WriteRune(rune('0' + spaceCount))
					spaceCount = 0
				}
				sb.WriteString(pc.String())
			}
		}
		if spaceCount != 0 {
			sb.WriteRune(rune('0' + spaceCount))
			spaceCount = 0
		}
		if y != 9 {
			sb.WriteRune('/')
		}
	}
	sb.WriteRune(' ')
	if pos.playerSd == SdRed {
		sb.WriteRune('r')
	} else {
		sb.WriteRune('b')
	}
	sb.WriteString(" - - 0 1")
	return sb.String()
}

// 创建局面
func CreatePosition() *Position {
	pos := &Position{}
	pos.playerSd = SdRed
	pos.mvStack = make([]historyMove, 1, limitDepth*2)
	return pos
}
func CreatePositionFromPosStr(positionStr string) (*Position, error) {
	return parsePosition(positionStr)
}
func CreatePositionFromFenStr(fenStr string) (*Position, error) {
	return parseFen(fenStr)
}
func revertSlice(mvs []Move) {
	for i, j := 0, len(mvs)-1; i < j; i, j = i+1, j-1 {
		mvs[i], mvs[j] = mvs[j], mvs[i]
	}
}
