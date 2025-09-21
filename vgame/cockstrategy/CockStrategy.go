package cock

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	com "vgame/_common"
)

type CockStrategy struct {
	server     *com.GameServer
	gameId     com.GameId
	name       string
	timeKeeper *com.TimeKeeper
	roomList   []*com.GameRoom

	roundId  com.RoundId
	tickTime float64

	resultNum int
	txh       string
	w         string
	stateMng  *com.StateManager
	// payout
	GameData   *CockStrategyData
	gameNumber com.GameNumber // need to gen on state STARTING

	pathIds []int
	pathInd int
	trends  []*com.TrendItem

	tickCount int
}

// work as constructor
func (g *CockStrategy) Init(server *com.GameServer) *CockStrategy {
	g.server = server
	g.gameId = com.IDCockStrategy
	g.name = "CockStrategy"
	g.roundId = 0
	g.timeKeeper = &com.TimeKeeper{}
	g.roomList = []*com.GameRoom{}
	gameStates := []com.GameState{com.GAME_STATE_STARTING, com.GAME_STATE_BETTING, com.GAME_STATE_CLOSE_BETTING, com.GAME_STATE_GEN_RESULT, com.GAME_STATE_RESULT, com.GAME_STATE_PAYOUT}
	stateTimes := []float64{1, 30, 3, 0, 10, 8.0} // 0 mean wait forever
	g.stateMng = (&com.StateManager{}).Init(gameStates, stateTimes, g.onEnterState, g.onExitState)
	g.tickTime = 0
	g.tickCount = 0
	g.resultNum = -1
	g.trends = []*com.TrendItem{}
	g.txh = ""
	g.w = ""

	pathLength := 24
	for i := 0; i < pathLength; i++ {
		g.pathIds = append(g.pathIds, i)
	}
	g.suffleArr()

	// payout
	g.GameData = (&CockStrategyData{}).init()
	return g
}

func (g *CockStrategy) suffleArr() {
	var t int = 0
	l := len(g.pathIds)
	for i := 0; i < l; i++ {
		ind := int(math.Floor(com.VUtils.GetRandFloat64() * (float64(l) - 0.01)))
		t = g.pathIds[i]
		g.pathIds[i] = g.pathIds[ind]
		g.pathIds[ind] = t
	}
}

func (g *CockStrategy) Start() {
	fmt.Println("CockStrategy start")
	trends := g.server.LoadTrends(g.gameId, 0)
	if trends != nil {
		g.trends = trends
	}
	if !g.LoadGameState() {
		g.stateMng.ResetState()
		g.onStartComplete()
	}
}

func (g *CockStrategy) onStartComplete() {
	fmt.Println("CockStrategy start complete")
	g.stateMng.Start()
	gameConf := com.GameServerConfig.GameConfigMap[g.gameId]
	com.VUtils.RepeatCall(g.Update, gameConf.FrameTime, 0, g.GetTimeKeeper())
}

func (g *CockStrategy) LoadGameState() bool {
	row := g.server.DB.QueryRow("SELECT gamenumber, roundid, state, statetime, result, tx, w, h FROM gamestate WHERE gameid=?", g.gameId)
	var gameNumber int
	var roundId int
	var state int
	var stateTime float64
	var result int
	var tx string
	var w string
	var h string
	err := row.Scan(&gameNumber, &roundId, &state, &stateTime, &result, &tx, &w, &h)
	if err != nil {
		return false
	}
	g.gameNumber = com.GameNumber(gameNumber)
	g.roundId = com.RoundId(roundId)
	g.stateMng.CurrState = com.GameState(state)
	g.stateMng.StateTime = stateTime
	g.resultNum = result
	g.txh = tx
	g.w = w
	return true
}

func (g *CockStrategy) SaveGameState() {
	resultStr := strconv.Itoa(g.resultNum)
	str := fmt.Sprintf("%d_%s_%d_%d_%s_%s_%s", g.gameNumber, g.gameId, g.roundId, g.stateMng.CurrState, resultStr, g.txh, g.w)
	hash := com.VUtils.HashString(str)
	_, err := g.server.DB.Exec("UPDATE gamestate SET state=?, statetime=?, result=?, tx=?, w=?, h=? WHERE gamenumber=?", g.stateMng.CurrState, g.stateMng.StateTime, resultStr, g.txh, g.w, hash, g.gameNumber)
	if err != nil {
		com.VUtils.PrintError(err)
		g.server.Maintenance()
		return
	}
}

func (game *CockStrategy) Update(dt float64) {
	game.stateMng.StateUpdate(dt)
	game.tickTime += dt
	if game.tickTime > 1.0 {
		game.onTick()
		game.tickTime -= 1.0
	}
}

func (g *CockStrategy) Stop() {
	fmt.Println("CockStrategy stop")
	g.stateMng.ResetState()
}

func (g *CockStrategy) GetTimeKeeper() *com.TimeKeeper {
	return g.timeKeeper
}

func (g *CockStrategy) GetGameId() com.GameId {
	return g.gameId
}

func (g *CockStrategy) GetName() string {
	return g.name
}

func (g *CockStrategy) GetRoundId() com.RoundId {
	return g.roundId
}

func (g *CockStrategy) GetGameNumber() com.GameNumber {
	return g.gameNumber
}

func (g *CockStrategy) GetTrends() []*com.TrendItem {
	return g.trends
}

func (g *CockStrategy) GetTrendsByPage(page uint32) []*com.TrendItem {
	start := page * uint32(com.TREND_PAGE_SIZE)
	end := start + uint32(com.TREND_PAGE_SIZE)
	if start >= uint32(len(g.trends)) {
		return []*com.TrendItem{}
	}
	if end > uint32(len(g.trends)) {
		end = uint32(len(g.trends))
	}
	return g.trends[start:end]
}

func (g *CockStrategy) GetBetKind(betType com.BetType) com.BetKind {
	return g.GameData.BetKindMap[string(betType)]
}

func (g *CockStrategy) GetAllBetLimits() []map[com.Currency]map[com.BetKind]*com.BetLimit {
	limits := []map[com.Currency]map[com.BetKind]*com.BetLimit{g.GameData.SmallLimitBetMap, g.GameData.MediumLimitBetMap}
	return limits
}

func (g *CockStrategy) GetBetLimit(level com.LimitLevel) map[com.Currency]map[com.BetKind]*com.BetLimit {
	switch level {
	case com.LIMIT_LEVEL_SMALL:
		return g.GameData.SmallLimitBetMap
	case com.LIMIT_LEVEL_MEDIUM:
		return g.GameData.MediumLimitBetMap
	}

	return g.GameData.SmallLimitBetMap
}

func (g *CockStrategy) GetCurState() com.GameState {
	return g.stateMng.CurrState
}

func (g *CockStrategy) GetPayout(betKind com.BetKind) com.Amount {
	return g.GameData.PayoutMap[betKind]
}

func (g *CockStrategy) onTick() {
	switch g.stateMng.CurrState {
	case com.GAME_STATE_BETTING:
		remainBettingTime := uint16(g.stateMng.StateDurs[1] - g.stateMng.StateTime)
		for _, room := range g.roomList {
			res := (&com.TickResponse{}).Init(room)
			res.Time = remainBettingTime
			res.RoomTotalBet = float64(room.GetRoomTotalBet())
			room.BroadcastMessage(res)
		}
	}
	g.tickCount++
	if g.tickCount > 0 {
		g.tickCount = 0
		for _, room := range g.roomList {
			room.NotifyRoomStats()
		}
	}
}

func (g *CockStrategy) onEnterState(state com.GameState) {
	// only broadcast for the users joined room already, so use room to broadcast insteak
	fmt.Println("onEnterState ", state)
	switch state {
	case com.GAME_STATE_STARTING:
		g.onEnterStarting()
	case com.GAME_STATE_BETTING:
		g.onEnterBetting()
	case com.GAME_STATE_CLOSE_BETTING:
		g.onEnterCloseBetting()
	case com.GAME_STATE_GEN_RESULT:
		g.onEnterGenResult()
	case com.GAME_STATE_RESULT:
		g.onEnterResult()
	case com.GAME_STATE_PAYOUT:
		g.onEnterPayout()
	}
	g.SaveGameState()
}

func (g *CockStrategy) onExitState(state com.GameState) {
}

func (g *CockStrategy) onEnterStarting() {
	fmt.Println("CockStrategy entering STARTING state")
	g.roundId++
	g.pathInd++

	if g.pathInd > len(g.pathIds)-1 {
		g.pathInd = 0
		g.suffleArr()
	}

	g.txh = ""
	g.w = ""
	g.tickTime = 0
	g.resultNum = -1
	if g.roundId > com.RoundId(com.MAX_ROUND) {
		g.roundId = 1
	}
	// create a new game state and delete the old one
	tx, err := g.server.DB.Begin()
	if err != nil {
		com.VUtils.PrintError(err)
		g.server.Maintenance()
		return
	}
	_, err2 := tx.Exec("DELETE FROM gamestate WHERE gameid=?", g.gameId)
	if err2 != nil {
		com.VUtils.PrintError(err2)
		return
	}
	response, err3 := tx.Exec("INSERT INTO gamestate(gameid, roundid, state, statetime, result) VALUES(?,?,?,?,?)", g.gameId, g.roundId, com.GAME_STATE_STARTING, 0, "")
	if err3 != nil {
		tx.Rollback()
		com.VUtils.PrintError(err3)
		return
	}

	gameNumber, err1 := response.LastInsertId()
	if err1 != nil {
		tx.Rollback()
		com.VUtils.PrintError(err1)
		return
	}

	err4 := tx.Commit()
	if err4 != nil {
		com.VUtils.PrintError(err4)
		g.server.Maintenance()
		return
	}
	// assign new game number ---
	oldGameNumber := g.gameNumber
	if oldGameNumber > 0 {
		gameConf := g.server.GetGameConf(g.gameId)
		for _, operatorId := range gameConf.OperatorIds {
			// remove old bettings from wallet operator
			param := com.VUtils.WalletLocalMessageUint64(operatorId, com.WCMD_CLEAR_BETTING, uint64(oldGameNumber))
			g.server.WalletConn.Send(param, func(vs *com.VSocket, requestId uint64, resData []byte) {
				res := com.BaseGameResponse{}
				err := json.Unmarshal(resData, &res)
				if err != nil {
					com.VUtils.PrintError(err)
					g.server.Maintenance()
					return
				}
				//fmt.Println("NEW GAME cleared betting id count ", res.IntVal, " gameNumber ", oldGameNumber)
			}, nil)
		}
	}
	g.gameNumber = com.GameNumber(gameNumber)
	//fmt.Println("NEW GAME == gameId ", g.gameId, " gameNumber ", g.gameNumber)
	// -------

	for _, room := range g.roomList {
		room.ResetBets()
		res := (&com.BaseGameResponse{}).Init(room, com.CMD_START_GAME)
		room.BroadcastMessage(res)
	}
}

func (g *CockStrategy) onEnterBetting() {
	fmt.Println("CockStrategy entering BETTING state")
	for _, room := range g.roomList {
		res := (&com.BaseGameResponse{}).Init(room, com.CMD_START_BET_SUCCEED)
		room.BroadcastMessage(res)
	}
}

func (g *CockStrategy) onEnterCloseBetting() {
	fmt.Println("CockStrategy entering CLOSE_BETTING state")
	for _, room := range g.roomList {
		res := (&com.BaseGameResponse{}).Init(room, com.CMD_STOP_BET_SUCCEED)
		room.BroadcastMessage(res)
	}
}

func (g *CockStrategy) onEnterGenResult() {
	fmt.Println("CockStrategy entering GEN_RESULT state")
	for _, room := range g.roomList {
		room.NotifyEndBetting()
	}
	g.genResult()
}

func (g *CockStrategy) onEnterResult() {
	fmt.Println("CockStrategy entering RESULT state")
	if g.resultNum < 0 {
		msg := fmt.Sprintf("game %s has no result when payout", g.gameId)
		com.VUtils.PrintError(errors.New(msg))
		g.server.Maintenance()
		return
	}
	// calculate payout & save DB but not send payout to player, payout should send when state payout start
	for _, room := range g.roomList {
		result := strconv.Itoa(g.resultNum) + "_" + strconv.Itoa(g.pathIds[g.pathInd])
		res := (&com.ClientGameResultResponse{}).Init(room, com.CMD_GAME_RESULT, result, g.txh, g.w)
		room.BroadcastMessage(res)
	}
	// run it on other thread
	go g.payout()
}

func (g *CockStrategy) onEnterPayout() {
	fmt.Println("CockStrategy entering PAYOUT state")
	// broadcast payout to users ---
	for _, room := range g.roomList {
		room.NotifyReward()
	}
}

func (g *CockStrategy) GetRemainStateTime() float64 {
	return g.stateMng.StateDurs[g.stateMng.CurrState] - g.stateMng.StateTime
}

func (g *CockStrategy) GetTotalBetTime() float64 {
	return g.stateMng.StateDurs[com.GAME_STATE_BETTING]
}

func (g *CockStrategy) payout() {
	if g.resultNum < 0 {
		msg := fmt.Sprintf("Game %s has no result when payout", g.gameId)
		com.VUtils.PrintError(errors.New(msg))
		g.server.Maintenance()
		return
	}

	if g.gameNumber == 0 {
		msg := fmt.Sprintf("Game %s has no gameNumber when payout", g.gameId)
		com.VUtils.PrintError(errors.New(msg))
		g.server.Maintenance()
		return
	}
	success := true
	for _, room := range g.roomList {
		if !g.payoutRoom(room) {
			success = false
		}
	}

	if !success {
		msg := fmt.Sprintf("payout not success for game %s", g.gameId)
		com.VUtils.PrintError(errors.New(msg))
		g.server.Maintenance()
		return
	}
	//fmt.Println("payout success for gamenumber", g.gameNumber)
}

func (g *CockStrategy) payoutRoom(room *com.GameRoom) bool {
	success := true
	for uid, betInfo := range room.BetInfosMap {
		if betInfo.ConfirmedBetState == nil || com.GetTotalBet(betInfo.ConfirmedBetState) == 0 {
			continue
		}
		// dont pay again if payed already
		if betInfo.Payedout > 0 {
			continue
		}
		totalPay := com.Amount(0)
		betDetail := ""
		for _, betPlace := range betInfo.ConfirmedBetState {
			betPay := com.Amount(0)
			isWin, has := g.GameData.betResultMap[string(betPlace.Type)][g.resultNum]
			if has && isWin {
				betKind := g.GameData.BetKindMap[string(betPlace.Type)]
				betPay = g.GameData.PayoutMap[betKind] * betPlace.Amount
				totalPay += betPay
			}
			if betDetail != "" {
				betDetail += ","
			}
			betDetail += string(betPlace.Type) + "_" + com.FormatAmount(betPlace.Amount) + "_" + com.FormatAmount(com.Amount(betPay))
		}
		betInfo.TotalPay = totalPay
		betInfo.Payedout = 1

		/* NOTE: for lamda capture -----------------------------------------
		+ Must declare a new local scope variable for each for loop (same as capture with 'let' in javascript)
		+ Dont capture the variable outside for loop scope: uid
		*/
		userId := uid
		// -------------------------------------------------------------------
		room.SaveBetting(betInfo, betDetail, totalPay, func(vs *com.VSocket, requestId uint64, resData []byte) {
			res := com.BettingResponse{}
			err := json.Unmarshal(resData, &res)
			if err != nil {
				com.VUtils.PrintError(err)
				room.Server.Maintenance()
				return
			}
			if res.ErrorCode > 0 {
				success = false
			}
			//userId := res.UserId // alternative way
			betInfo := room.BetInfosMap[userId]
			betInfo.Balance = res.Balance
		})

	}
	return success
}

func (g *CockStrategy) GetGameResult() string {
	if g.resultNum < 0 {
		return ""
	}
	return fmt.Sprintf("%d", g.resultNum)
}

func (g *CockStrategy) GetResultString() string {
	return fmt.Sprintf("%d_%d", g.resultNum, g.pathIds[g.pathInd])
}

func (g *CockStrategy) GetTxh() string {
	return g.txh
}

func (g *CockStrategy) GetW() string {
	return g.w
}

func (g *CockStrategy) InitRoomForGame(room *com.GameRoom) {
	g.roomList = append(g.roomList, room)
	// init limit map
}

func (g *CockStrategy) LoadTrends(gameId com.GameId, page uint32) []*com.TrendItem {
	return g.trends
}

func (g *CockStrategy) OnMessage(connId com.ConnectionId, msg string) {
	//fmt.Printf("game: %s OnMessage: %s\n", g.name, msg)
	connInfo, ok := g.server.GetConnectionInfo(connId)
	if !ok {
		return
	}
	data := map[string]interface{}{}
	err := json.Unmarshal([]byte(msg), &data)
	if err != nil {
		return
	}

	cmd := data["CMD"].(string)
	roomId := data["RoomId"].(string)
	room := g.server.RoomMng.GetRoom(connInfo.OperatorId, com.RoomId(roomId))
	if room == nil {
		fmt.Println("OnMessage roomId == nil ", cmd)
		return
	}
	currState := g.stateMng.CurrState
	switch cmd {
	case com.CMD_SEND_BET_UPDATE:
		if currState == com.GAME_STATE_BETTING || currState == com.GAME_STATE_CLOSE_BETTING {
			// BetRequest
			betTypes := data["BetTypes"].([]interface{})
			amounts := data["Amounts"].([]interface{})
			betState := []*com.BetPlace{}
			for i, v := range betTypes {
				betType := v.(string)
				amount := amounts[i].(float64)
				betPlace := &com.BetPlace{Type: com.BetType(betType), Amount: com.Amount(amount)}
				betState = append(betState, betPlace)
			}
			room.ProcessBets(connId, betState)
		} else {
			res := (&com.BetFailResponse{}).Init(room)
			res.FailCode = com.RES_FAIL_BET_REJECT
			g.server.SendPrivateMessage(connId, res)
		}
	}

	room.OnMessage(cmd, connInfo, msg)
}

func (g *CockStrategy) genResult() {
	// Empty implementation - will be implemented later
	fmt.Println("CockStrategy genResult() - empty implementation")
}
