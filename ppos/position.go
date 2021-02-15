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

// 空着裁剪阈值
const nullPruneThreshold = 400

// 空着裁剪深度
const nullPruneDepth = 2

type searchCtx struct {
	// 电脑走的棋
	mvResult Move

	// 已搜索的局面数
	nPositionCount int

	// 停止搜索的时间
	stopSearchTime time.Time

	// 停止搜索
	stopSearch bool

	// 在空着搜索过程中
	inNullMoveSearch bool

	// 开始搜索时局面的distance
	initDistance int

	// 此次搜索最大距离
	maxDistance int

	// 历史表
	historyTable [65536]int
}

type HistoryMove struct {
	move       Move
	pcCaptured Piece
	checked    bool
	posZobrist ZobristHash
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
	mvStack []HistoryMove

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
	if pos.Checked() {
		pos.UndoMovePiece(move, pcCaptured)
		return false
	}
	pos.ChangeSide()
	pos.mvStack = append(pos.mvStack, HistoryMove{move, pcCaptured, pos.Checked(), preZob})
	pos.nDistance++
	return true
}

// 等效 MakeMove(MvNop) 但是更快
func (pos *Position) makeNullMove() {
	pos.ChangeSide()
	pos.mvStack = append(pos.mvStack, HistoryMove{MvNop, PcNop, pos.InCheck(), pos.zobrist})
	pos.nDistance++
}
func (pos *Position) UndoMakeMove() {
	pos.ChangeSide()
	moveHis := pos.mvStack[pos.nDistance]
	pos.UndoMovePiece(moveHis.move, moveHis.pcCaptured)
	pos.nDistance--
	pos.mvStack = pos.mvStack[:pos.nDistance+1]
}

// 等效 UndoMakeMove() 但是更快
func (pos *Position) undoNullMove() {
	pos.ChangeSide()
	pos.nDistance--
	pos.mvStack = pos.mvStack[:pos.nDistance+1]
}

// 当前局面能否空着裁剪
func (pos *Position) nullOk() bool {
	if pos.playerSd == SdRed {
		return pos.vlRed > nullPruneThreshold
	} else {
		return pos.vlBlack > nullPruneThreshold
	}
}
func (pos *Position) searchAlphaBeta(ctx *searchCtx, vlAlpha, vlBeta, depth int) int {
	tickSearch(ctx)
	if pos.nDistance > ctx.initDistance {
		// 1. 到达水平线，使用静态局面搜索
		if depth <= 0 {
			return pos.searchQuiescent(ctx, vlAlpha, vlBeta)
		}

		// 1-1. 检查重复局面
		rep, vl := pos.CheckReputation()
		if rep {
			return vl
		}

		// 1-2. 到达极限深度就返回局面评价
		if pos.nDistance == ctx.maxDistance {
			return pos.Evaluate()
		}

		// 1-3. 尝试空步裁剪
		if !ctx.inNullMoveSearch && !pos.InCheck() && pos.nullOk() {
			ctx.inNullMoveSearch = true
			pos.makeNullMove()
			// 窗口缩小
			vl = -pos.searchAlphaBeta(ctx, -vlBeta, 1-vlBeta, depth-1-nullPruneDepth)
			pos.undoNullMove()
			ctx.inNullMoveSearch = false
			if vl >= vlBeta {
				return vl
			}
		}
	}

	moves := make([]Move, 0, initMovesSize)
	vlBest := -mateValue
	var mvBest = MvNop
	moves = pos.GenerateMoves(moves, false)
	sort.Sort(MoveSorter{moves: moves, eval: func(mv Move) int {
		return ctx.historyTable[mv]
	}})
	for _, mv := range moves {
		if !pos.MakeMove(mv) {
			continue
		}
		vl := -pos.searchAlphaBeta(ctx, -vlBeta, -vlAlpha, depth-1)
		pos.UndoMakeMove()
		if ctx.stopSearch {
			return 0
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
			}
		}
	}
	// 所有的move都无法走 杀棋!
	if vlBest == -mateValue {
		return pos.nDistance - ctx.initDistance - mateValue
	}
	if mvBest != MvNop {
		ctx.historyTable[mvBest] += depth * depth
		if pos.nDistance == ctx.initDistance {
			ctx.mvResult = mvBest
		}
	}
	return vlBest
}

// 搜索中的状态检查
func tickSearch(searchCtx *searchCtx) {
	searchCtx.nPositionCount++
	if searchCtx.nPositionCount&0x1fff == 0 && time.Now().After(searchCtx.stopSearchTime) {
		searchCtx.stopSearch = true
	}
}
func (pos *Position) searchQuiescent(ctx *searchCtx, vlAlpha, vlBeta int) int {
	// 1. 检查重复局面
	rep, vl := pos.CheckReputation()
	if rep {
		return vl
	}

	// 2. 到达极限深度就返回局面评价
	if pos.nDistance == ctx.maxDistance {
		return pos.Evaluate()
	}

	// 3. 初始化最佳值
	vlBest := -mateValue

	moves := make([]Move, 0, initMovesSize)
	if pos.InCheck() {
		// 4. 如果被将军，则生成全部走法
		moves = pos.GenerateMoves(moves, false)
		sort.Sort(MoveSorter{moves: moves, eval: func(mv Move) int {
			return ctx.historyTable[mv]
		}})
	} else {
		// 5. 如果不被将军，先做局面评价
		vl = pos.Evaluate()
		if vl > vlBest {
			vlBest = vl
			if vl >= vlBeta {
				return vl
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
		vl = -pos.searchQuiescent(ctx, -vlBeta, -vlAlpha)
		pos.UndoMakeMove()

		if vl > vlBest {
			vlBest = vl
			if vl >= vlBeta {
				return vl
			}
			if vl > vlAlpha {
				vlAlpha = vl
			}
		}
	}

	if vlBest == -mateValue {
		return pos.nDistance - ctx.initDistance - mateValue
	} else {
		return vlBest
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
func (pos *Position) SearchMain(duration time.Duration) (Move, int) {
	startTime := time.Now()
	ctx := &searchCtx{}
	ctx.stopSearchTime = time.Now().Add(duration)
	ctx.initDistance = pos.nDistance
	ctx.maxDistance = pos.nDistance + limitDepth
	vl := 0
	bestMove := MvNop
	nPositions := 0
	maxDepth := 0
	for ; maxDepth < limitDepth; maxDepth++ {
		ctx.nPositionCount = 0
		res := pos.searchAlphaBeta(ctx, -mateValue, mateValue, maxDepth)
		if ctx.stopSearch {
			maxDepth--
			break
		}
		vl = res
		bestMove = ctx.mvResult
		nPositions = ctx.nPositionCount
		if vl > winValue || vl < -winValue {
			break
		}
	}
	logrus.Infof("search depth: %d, search nodes: %d, search time: %v, best move: %v", maxDepth, nPositions, time.Now().Sub(startTime), bestMove)
	return bestMove, vl
}
func (pos *Position) String() string {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		if i == 4 || i == 5 {
			sb.WriteRune('-')
		} else {
			sb.WriteRune(' ')
		}
		sb.WriteString(pos.pcSquares[GetSquare(3, i+3)].String())
		for j := 1; j < 9; j++ {
			sb.WriteRune('-')
			sb.WriteString(pos.pcSquares[GetSquare(j+3, i+3)].String())
		}
		if i == 4 || i == 5 {
			sb.WriteRune('-')
		} else {
			sb.WriteRune(' ')
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

// 创建局面
func CreatePosition() *Position {
	pos := &Position{}
	pos.playerSd = SdRed
	pos.mvStack = make([]HistoryMove, 1, limitDepth*2)
	return pos
}

var lineMoveDelta = [4]Square{-0x10, -0x01, +0x01, +0x10}
var advisorMoveTab = [4]Square{-0x11, -0x0f, +0x0f, +0x11}
var bishopMoveTab = [4]Square{-0x22, -0x1e, +0x1e, +0x22}
var knightMoveTab = [8]Square{-0x21, -0x1f, -0x12, -0x0e, +0x0e, +0x12, +0x1f, +0x21}
var knightMovePinTab = [512]Square{
	0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, -0x10, 0, -0x10, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, -0x01, 0, 0, 0, +0x01, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, -0x01, 0, 0, 0, +0x01, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0x10, 0, 0x10, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0,
}

func getKnightPin(sqSrc Square, sqDst Square) Square {
	return sqSrc + knightMovePinTab[256+sqDst-sqSrc]
}
func sqForward(sq Square, sd Side) Square {
	return sq + Square((sd>>1)<<5-0x10)
}

// 子力位置价值表
var pieceValue = [7][256]int{
	{ // 帅(将)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 11, 15, 11, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 仕(士)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 20, 0, 20, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 23, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 20, 0, 20, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 相(象)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 20, 0, 0, 0, 20, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 18, 0, 0, 0, 23, 0, 0, 0, 18, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 20, 0, 0, 0, 20, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 马
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 90, 90, 90, 96, 90, 96, 90, 90, 90, 0, 0, 0, 0,
		0, 0, 0, 90, 96, 103, 97, 94, 97, 103, 96, 90, 0, 0, 0, 0,
		0, 0, 0, 92, 98, 99, 103, 99, 103, 99, 98, 92, 0, 0, 0, 0,
		0, 0, 0, 93, 108, 100, 107, 100, 107, 100, 108, 93, 0, 0, 0, 0,
		0, 0, 0, 90, 100, 99, 103, 104, 103, 99, 100, 90, 0, 0, 0, 0,
		0, 0, 0, 90, 98, 101, 102, 103, 102, 101, 98, 90, 0, 0, 0, 0,
		0, 0, 0, 92, 94, 98, 95, 98, 95, 98, 94, 92, 0, 0, 0, 0,
		0, 0, 0, 93, 92, 94, 95, 92, 95, 94, 92, 93, 0, 0, 0, 0,
		0, 0, 0, 85, 90, 92, 93, 78, 93, 92, 90, 85, 0, 0, 0, 0,
		0, 0, 0, 88, 85, 90, 88, 90, 88, 90, 85, 88, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 车
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 206, 208, 207, 213, 214, 213, 207, 208, 206, 0, 0, 0, 0,
		0, 0, 0, 206, 212, 209, 216, 233, 216, 209, 212, 206, 0, 0, 0, 0,
		0, 0, 0, 206, 208, 207, 214, 216, 214, 207, 208, 206, 0, 0, 0, 0,
		0, 0, 0, 206, 213, 213, 216, 216, 216, 213, 213, 206, 0, 0, 0, 0,
		0, 0, 0, 208, 211, 211, 214, 215, 214, 211, 211, 208, 0, 0, 0, 0,
		0, 0, 0, 208, 212, 212, 214, 215, 214, 212, 212, 208, 0, 0, 0, 0,
		0, 0, 0, 204, 209, 204, 212, 214, 212, 204, 209, 204, 0, 0, 0, 0,
		0, 0, 0, 198, 208, 204, 212, 212, 212, 204, 208, 198, 0, 0, 0, 0,
		0, 0, 0, 200, 208, 206, 212, 200, 212, 206, 208, 200, 0, 0, 0, 0,
		0, 0, 0, 194, 206, 204, 212, 200, 212, 204, 206, 194, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 炮
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 100, 100, 96, 91, 90, 91, 96, 100, 100, 0, 0, 0, 0,
		0, 0, 0, 98, 98, 96, 92, 89, 92, 96, 98, 98, 0, 0, 0, 0,
		0, 0, 0, 97, 97, 96, 91, 92, 91, 96, 97, 97, 0, 0, 0, 0,
		0, 0, 0, 96, 99, 99, 98, 100, 98, 99, 99, 96, 0, 0, 0, 0,
		0, 0, 0, 96, 96, 96, 96, 100, 96, 96, 96, 96, 0, 0, 0, 0,
		0, 0, 0, 95, 96, 99, 96, 100, 96, 99, 96, 95, 0, 0, 0, 0,
		0, 0, 0, 96, 96, 96, 96, 96, 96, 96, 96, 96, 0, 0, 0, 0,
		0, 0, 0, 97, 96, 100, 99, 101, 99, 100, 96, 97, 0, 0, 0, 0,
		0, 0, 0, 96, 97, 98, 98, 98, 98, 98, 97, 96, 0, 0, 0, 0,
		0, 0, 0, 96, 96, 97, 99, 99, 99, 97, 96, 96, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 兵(卒)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 9, 9, 9, 11, 13, 11, 9, 9, 9, 0, 0, 0, 0,
		0, 0, 0, 19, 24, 34, 42, 44, 42, 34, 24, 19, 0, 0, 0, 0,
		0, 0, 0, 19, 24, 32, 37, 37, 37, 32, 24, 19, 0, 0, 0, 0,
		0, 0, 0, 19, 23, 27, 29, 30, 29, 27, 23, 19, 0, 0, 0, 0,
		0, 0, 0, 14, 18, 20, 27, 29, 27, 20, 18, 14, 0, 0, 0, 0,
		0, 0, 0, 7, 0, 13, 0, 16, 0, 13, 0, 7, 0, 0, 0, 0,
		0, 0, 0, 7, 0, 7, 0, 15, 0, 7, 0, 7, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	},
}

// mvv 子力价值
var mvvLvaPieceValue = []int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 1, 1, 3, 4, 3, 2, 0,
	5, 1, 1, 3, 4, 3, 2, 0,
}
