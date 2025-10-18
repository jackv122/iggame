package cock

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	com "vgame/_common"
)

type CockStrategy struct {
	com.BaseGame
	// payout
	GameData       *CockStrategyData
	gameStateData  *GameStateData
	gameResultData *GameResultData

	battleConfig BattleConfig
	pairIndex    int
}

type CockData struct {
	Name     string
	ID       CockID
	Strength float64
	Agility  float64
	Payout   com.Amount
}

type GenResultContent struct {
	Cock1   *CockData
	Cock2   *CockData
	Randoms []string
}

// work as constructor
func (g *CockStrategy) Init(server *com.GameServer) *CockStrategy {
	g.InitBase(server, com.IDCockStrategy, "CockStrategy")
	gameStates := []com.GameState{com.GAME_STATE_STARTING, com.GAME_STATE_BETTING, com.GAME_STATE_CLOSE_BETTING, com.GAME_STATE_GEN_RESULT, com.GAME_STATE_RESULT, com.GAME_STATE_PAYOUT}
	stateTimes := []float64{1, 20, 3, 0, float64(com.MAX_PAYOUT_WAIT_TIME), 5.0} // 0 mean wait forever
	g.StateMng = (&com.StateManager{}).Init(gameStates, stateTimes, g.onEnterState, g.onExitState)
	g.GameData = (&CockStrategyData{}).init(g)
	g.gameResultData = nil

	g.gameStateData = &GameStateData{
		Version: GAME_VERSION,
	}

	// parse battle configs from config/cock_strategy/
	dirPath := fmt.Sprintf("config/cock_strategy/%s/", GAME_VERSION)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		com.VUtils.PrintError(err)
		panic("Failed to load battle configs: " + err.Error())
	}
	g.GameData.BattleConfigs = make([]BattleConfig, 0)
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		// load stats
		filePath := path.Join(dirPath, file.Name(), "stats.json")
		content, err := os.ReadFile(filePath)
		if err != nil {
			com.VUtils.PrintError(err)
			continue
		}
		var battleConfig BattleConfig = BattleConfig{}
		var Stats Stats = Stats{}
		err = json.Unmarshal(content, &Stats)
		if err != nil {
			com.VUtils.PrintError(err)
			continue
		}
		battleConfig.Stats = Stats

		// load db
		filePath = path.Join(dirPath, file.Name(), "db.txt")
		content, err = os.ReadFile(filePath)
		if err != nil {
			com.VUtils.PrintError(err)
			continue
		}
		var db []string = strings.Split(string(content), "\n")
		battleConfig.DB = db
		g.GameData.BattleConfigs = append(g.GameData.BattleConfigs, battleConfig)
	}
	return g
}

func (g *CockStrategy) Start() {
	fmt.Printf("%s start\n", g.Name)
	trends := g.Server.LoadTrends(g.GameId, 0)
	if trends != nil {
		g.Trends = trends
	}
	if !g.LoadGameState() {
		g.StateMng.ResetState()
		g.OnStartComplete()
	}
}

func (g *CockStrategy) OnStartComplete() {
	fmt.Println("CockStrategy start complete")
	g.StateMng.Start()
	gameConf := com.GameServerConfig.GameConfigMap[g.GameId]
	com.VUtils.RepeatCall(g.Update, gameConf.FrameTime, 0, g.GetTimeKeeper())
}

func (g *CockStrategy) LoadGameState() bool {
	row := g.Server.DB.QueryRow("SELECT gamenumber, roundid, state, statetime, result, data, tx, w, h FROM gamestate WHERE gameid=?", g.GameId)

	var currState = com.GameState(com.GAME_STATE_STARTING)
	result := ""
	gameDataStr := ""
	hash := ""
	statetime := float64(0)
	err := row.Scan(&g.GameNumber, &g.RoundId, &currState, &statetime, &result, &gameDataStr, &g.Txh, &g.W, &hash)
	if err != nil {
		fmt.Println("not existing previous state data for game ", g.GameId)
		return false
	}
	str := fmt.Sprintf("%d_%s_%d_%d_%s_%s_%s", g.GameNumber, g.GameId, g.RoundId, currState, result, g.Txh, g.W)
	//fmt.Println("LoadGameState", str, com.VUtils.HashString(str), "hash", hash)
	if hash != com.VUtils.HashString(str) {
		fmt.Println("Wrong hash for game ", g.GameId)
		g.Server.Stop()
		return false
	}

	g.gameResultData = nil

	if result != "" {
		err = json.Unmarshal([]byte(result), &g.gameResultData)
		if err != nil {
			msg := fmt.Sprintf("can not parse game result for gameId %s and result %s ", g.GameId, result)
			com.VUtils.PrintError(errors.New(msg))
			g.Server.Maintenance()
			return true
		}
	}

	g.gameStateData = nil

	if gameDataStr != "" {
		err = json.Unmarshal([]byte(gameDataStr), &g.gameStateData)
		if err != nil {
			msg := fmt.Sprintf("can not parse game data for gameId %s and result %s ", g.GameId, gameDataStr)
			com.VUtils.PrintError(errors.New(msg))
			g.Server.Maintenance()
			return true
		}
	}

	// resume state
	g.StateMng.SetState(currState, statetime)
	g.battleConfig = g.GameData.BattleConfigs[g.gameStateData.PairIndex]
	g.pairIndex = g.gameStateData.PairIndex

	// reload all bettings
	gameConf := com.GameServerConfig.GameConfigMap[g.GameId]
	loadRequestCount := len(gameConf.OperatorIds)
	for _, operatorId := range gameConf.OperatorIds {
		param := com.VUtils.WalletLocalMessageUint64(operatorId, com.WCMD_QUERY_BETTING, uint64(g.GameNumber))
		g.Server.WalletConn.Send(param, func(vs *com.VSocket, requestId uint64, resData []byte) {
			res := com.QueryBettingResponse{}
			err := json.Unmarshal(resData, &res)
			if err != nil {
				com.VUtils.PrintError(err)
				g.Server.Maintenance()
				return
			}
			for _, roomConf := range gameConf.RoomConfigs {
				room := g.Server.RoomMng.GetRoom(operatorId, roomConf.RoomId)
				room.ResumeBetting(res.Bettings)
			}
			fmt.Println("loadRequestCount ===")
			loadRequestCount--
			if loadRequestCount == 0 {
				g.OnStartComplete()
			}
		}, nil)
	}

	return true
}

func (g *CockStrategy) SaveGameState() {
	gameDataStr, err := json.Marshal(g.gameStateData)
	if err != nil {
		com.VUtils.PrintError(err)
		return
	}

	resultStr, err := json.Marshal(g.gameResultData)
	if err != nil {
		com.VUtils.PrintError(err)
		return
	}

	str := fmt.Sprintf("%d_%s_%d_%d_%s_%s_%s", g.GameNumber, g.GameId, g.RoundId, g.StateMng.CurrState, resultStr, g.Txh, g.W)
	hash := com.VUtils.HashString(str)
	//fmt.Println("SaveGameState", str, com.VUtils.HashString(str), "hash", hash)
	_, err2 := g.Server.DB.Exec("UPDATE gamestate SET state=?, statetime=?, result=?, data=?, tx=?, w=?, h=? WHERE gamenumber=?", g.StateMng.CurrState, g.StateMng.StateTime, resultStr, gameDataStr, g.Txh, g.W, hash, g.GameNumber)
	if err2 != nil {
		com.VUtils.PrintError(err2)
		g.Server.Maintenance()
		return
	}
}

func (g *CockStrategy) onEnterState(state com.GameState) {
	// only broadcast for the users joined room already, so use room to broadcast insteak
	fmt.Println("onEnterState ", state)
	switch state {
	case com.GAME_STATE_STARTING:
		g.OnEnterStarting()
	case com.GAME_STATE_BETTING:
		g.OnEnterBetting()
	case com.GAME_STATE_CLOSE_BETTING:
		g.OnEnterCloseBetting()
	case com.GAME_STATE_GEN_RESULT:
		g.OnEnterGenResult()
	case com.GAME_STATE_RESULT:
		g.OnEnterResult()
	case com.GAME_STATE_PAYOUT:
		g.OnEnterPayout()
	}
	g.SaveGameState()
}

func (g *CockStrategy) onExitState(state com.GameState) {
}

func (g *CockStrategy) GetBetKind(betType com.BetType) com.BetKind {
	return g.BetKindMap[string(betType)]
}

func (g *CockStrategy) GetAllBetLimits() []map[com.Currency]map[com.BetKind]*com.BetLimit {
	limits := []map[com.Currency]map[com.BetKind]*com.BetLimit{g.SmallLimitBetMap, g.MediumLimitBetMap}
	return limits
}

func (g *CockStrategy) GetBetLimit(level com.LimitLevel) map[com.Currency]map[com.BetKind]*com.BetLimit {
	switch level {
	case com.LIMIT_LEVEL_SMALL:
		return g.SmallLimitBetMap
	case com.LIMIT_LEVEL_MEDIUM:
		return g.MediumLimitBetMap
	}

	return g.SmallLimitBetMap
}

func (g *CockStrategy) GetPayout(betKind com.BetKind) com.Amount {
	return g.PayoutMap[string(betKind)]
}

func (g *CockStrategy) OnEnterStarting() {
	fmt.Println("CockStrategy entering STARTING state")
	g.RoundId++

	g.Txh = ""
	g.W = ""
	g.TickTime = 0
	g.gameResultData = nil
	if g.RoundId > com.RoundId(com.MAX_ROUND) {
		g.RoundId = 1
	}
	// create a new game state and delete the old one
	tx, err := g.Server.DB.Begin()
	if err != nil {
		com.VUtils.PrintError(err)
		g.Server.Maintenance()
		return
	}
	_, err2 := tx.Exec("DELETE FROM gamestate WHERE gameid=?", g.GameId)
	if err2 != nil {
		com.VUtils.PrintError(err2)
		return
	}
	g.pairIndex++
	if g.pairIndex >= len(g.GameData.BattleConfigs) {
		g.pairIndex = 0
	}
	g.battleConfig = g.GameData.BattleConfigs[g.pairIndex]
	stats := g.battleConfig.Stats
	left := stats.LeftCockConfig
	right := stats.RightCockConfig
	fee := 0.04
	leftPayout := float64(stats.Total)/float64(stats.Win[string(left.ID)]) - fee
	rightPayout := float64(stats.Total)/float64(stats.Win[string(right.ID)]) - fee

	g.gameStateData.Cock_1 = &CockData{
		Name:     left.Name,
		ID:       left.ID,
		Strength: float64(left.S),
		Agility:  float64(left.A),
		Payout:   com.Amount(leftPayout),
	}
	g.gameStateData.Cock_2 = &CockData{
		Name:     right.Name,
		ID:       right.ID,
		Strength: float64(right.S),
		Agility:  float64(right.A),
		Payout:   com.Amount(rightPayout),
	}
	g.gameStateData.PairIndex = g.pairIndex
	g.gameStateData.ResultBattleIndex = -1

	g.PayoutMap = map[string]com.Amount{
		string(BET_TYPE_LEFT):  g.gameStateData.Cock_1.Payout,
		string(BET_TYPE_RIGHT): g.gameStateData.Cock_2.Payout,
	}

	gameDataStr, err := json.Marshal(&g.gameStateData)
	if err != nil {
		com.VUtils.PrintError(err)
		return
	}
	response, err3 := tx.Exec("INSERT INTO gamestate(gameid, roundid, state, statetime, result, data) VALUES(?,?,?,?,?,?)", g.GameId, g.RoundId, com.GAME_STATE_STARTING, 0, "", gameDataStr)
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
		g.Server.Maintenance()
		return
	}
	// assign new game number ---
	oldGameNumber := g.GameNumber
	if oldGameNumber > 0 {
		gameConf := g.Server.GetGameConf(g.GameId)
		for _, operatorId := range gameConf.OperatorIds {
			// remove old bettings from wallet operator
			param := com.VUtils.WalletLocalMessageUint64(operatorId, com.WCMD_CLEAR_BETTING, uint64(oldGameNumber))
			g.Server.WalletConn.Send(param, func(vs *com.VSocket, requestId uint64, resData []byte) {
				res := com.BaseWalletResponse{}
				err := json.Unmarshal(resData, &res)
				if err != nil {
					com.VUtils.PrintError(err)
					g.Server.Maintenance()
					return
				}
				//fmt.Println("NEW GAME cleared betting id count ", res.IntVal, " gameNumber ", oldGameNumber)
			}, nil)
		}
	}
	g.GameNumber = com.GameNumber(gameNumber)
	//fmt.Println("NEW GAME == gameId ", g.GameId, " gameNumber ", g.GameNumber)
	// -------

	for _, room := range g.RoomList {
		room.ResetBets()
		room.GameInitData = g.gameStateData.GetInitData()
		res := (&com.BaseGameResponse{}).Init(room, com.CMD_START_GAME)
		res.Data = g.gameStateData
		room.BroadcastMessage(res)
	}
}

func (g *CockStrategy) OnEnterBetting() {
	g.BaseGame.OnEnterBetting()
}

func (g *CockStrategy) OnEnterCloseBetting() {
	g.BaseGame.OnEnterCloseBetting()
}

func (g *CockStrategy) OnEnterGenResult() {
	fmt.Printf("%s entering GEN_RESULT state\n", g.Name)
	for _, room := range g.RoomList {
		room.NotifyEndBetting()
	}
	g.genResult()
}

func (g *CockStrategy) OnEnterResult() {
	fmt.Println("CockStrategy entering RESULT state")

	if g.gameResultData == nil {
		msg := fmt.Sprintf("game %s has no result when payout", g.GameId)
		com.VUtils.PrintError(errors.New(msg))
		g.Server.Maintenance()
		return
	}

	// calculate payout & save DB but not send payout to player, payout should send when state payout start
	for _, room := range g.RoomList {
		if g.gameResultData != nil {
			if len(g.gameResultData.HighlightGates) > 1 {
				g.Server.Maintenance()
				panic("HighlightGates > 1 " + fmt.Sprintf("%v", g.gameResultData.HighlightGates))
			}
		}
		res := (&com.ClientGameResultResponse{}).Init(room, com.CMD_GAME_RESULT, g.gameResultData, g.Txh, g.W)
		room.BroadcastMessage(res)
	}
	// run it on other thread
	go g.payout()
}

func (g *CockStrategy) OnEnterPayout() {
	g.BaseGame.OnEnterPayout()
}

func (g *CockStrategy) payout() {
	if g.gameResultData == nil {
		msg := fmt.Sprintf("Game %s has no result when payout", g.GameId)
		com.VUtils.PrintError(errors.New(msg))
		g.Server.Maintenance()
		return
	}

	if g.GameNumber == 0 {
		msg := fmt.Sprintf("Game %s has no gameNumber when payout", g.GameId)
		com.VUtils.PrintError(errors.New(msg))
		g.Server.Maintenance()
		return
	}
	success := true
	for _, room := range g.RoomList {
		if !g.payoutRoom(room) {
			success = false
		}
	}

	if !success {
		msg := fmt.Sprintf("payout not success for game %s", g.GameId)
		com.VUtils.PrintError(errors.New(msg))
		g.Server.Maintenance()
		return
	}
	g.StateMng.NextState()
	//fmt.Println("payout success for gamenumber", g.GameNumber)
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
		confirmPayouts := []*com.PayoutInfo{}
		for _, betPlace := range betInfo.ConfirmedBetState {
			betPay := com.Amount(0)

			isWin := g.GameData.betResultMap[betPlace.Type]
			if isWin {
				betPay = g.PayoutMap[string(betPlace.Type)] * betPlace.Amount
				totalPay += betPay
			}

			// apply payout also for not win bet
			confirmPayouts = append(confirmPayouts, &com.PayoutInfo{BetType: betPlace.Type, BetAmount: betPlace.Amount, PayoutAmount: com.TruncateAmount(betPay)})

			if betDetail != "" {
				betDetail += ","
			}
			betDetail += string(betPlace.Type) + "_" + com.FormatAmount(betPlace.Amount) + "_" + com.FormatAmount(com.Amount(betPay))
		}
		betInfo.TotalPay = totalPay
		betInfo.Payedout = 1
		betInfo.ConfirmedPayouts = confirmPayouts

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
				// error
				room.Server.Maintenance()
				return
			}
			//userId := res.UserId // alternative way
			betInfo := room.BetInfosMap[userId]

			betInfo.Balance = res.Balance
		})

	}
	return success
}

func (g *CockStrategy) OnMessage(roomId com.RoomId, connId com.ConnectionId, msg string) {
	//fmt.Printf("game: %s OnMessage: %s\n", g.Name, msg)
	connInfo, ok := g.Server.GetConnectionInfo(connId)
	if !ok {
		return
	}
	data := map[string]interface{}{}
	err := json.Unmarshal([]byte(msg), &data)
	if err != nil {
		return
	}

	cmd := data["CMD"].(string)
	//roomId := data["RoomId"].(string)
	room := g.Server.RoomMng.GetRoom(connInfo.OperatorId, com.RoomId(roomId))
	if room == nil {
		fmt.Println("OnMessage roomId == nil ", cmd)
		return
	}
	currState := g.StateMng.CurrState
	switch cmd {
	case com.CMD_SEND_BET_UPDATE:
		if currState == com.GAME_STATE_BETTING || currState == com.GAME_STATE_CLOSE_BETTING {
			// BetRequest
			clientRequestId := data["ClientRequestId"].(string)
			betTypes := data["BetTypes"].([]interface{})
			amounts := data["Amounts"].([]interface{})
			betState := []*com.BetPlace{}
			for i, v := range betTypes {
				betType := v.(string)
				amount := amounts[i].(float64)
				betPlace := &com.BetPlace{Type: com.BetType(betType), Amount: com.Amount(amount)}
				betState = append(betState, betPlace)
			}
			room.ProcessBets(connId, clientRequestId, betState)
		} else {
			res := (&com.BetFailResponse{}).Init(room)
			res.FailCode = com.RES_FAIL_BET_REJECT
			g.Server.SendPrivateMessage(room.RoomId, connId, res)
		}
	}

	room.OnMessage(cmd, connInfo, msg)
}

func (g *CockStrategy) GetResultData() interface{} {
	return g.gameResultData
}

func (game *CockStrategy) GetGameResultString() string {
	if game.gameResultData == nil {
		return ""
	}
	return string(game.gameResultData.Winner)
}

func (g *CockStrategy) genResult() {

	betTypes := []com.BetType{
		BET_TYPE_LEFT,
		BET_TYPE_RIGHT}

	// TODO: must modify the rand for fair and safe
	l := len(g.battleConfig.DB)

	battleIndex := rand.Intn(l)
	// Resume game
	if g.gameStateData.ResultBattleIndex > -1 {
		battleIndex = g.gameStateData.ResultBattleIndex
	}
	g.gameStateData.ResultBattleIndex = battleIndex

	battleInfoStr := g.battleConfig.DB[battleIndex]
	// parse battleInfo json
	var battleInfo BattleInfo = BattleInfo{}
	err := json.Unmarshal([]byte(battleInfoStr), &battleInfo)
	if err != nil {
		com.VUtils.PrintError(err)
		g.Server.Maintenance()
		return
	}

	betTypeToCockMap := map[com.BetType]CockID{
		BET_TYPE_LEFT:  g.gameStateData.Cock_1.ID,
		BET_TYPE_RIGHT: g.gameStateData.Cock_2.ID,
	}

	winner := CockID(battleInfo.Winner)

	highlightGates := []com.BetType{}
	// dynamic update betResultMap base on game result
	for _, betType := range betTypes {
		if winner == betTypeToCockMap[betType] {
			g.GameData.betResultMap[betType] = true

			// marks winner bet type as highlight gate
			highlightGates = append(highlightGates, betType)
		} else {
			g.GameData.betResultMap[betType] = false
		}
	}

	// debug
	if len(highlightGates) > 1 {
		g.Server.Maintenance()
	}
	// ------------------------------------------------------------

	g.gameResultData = &GameResultData{
		Version:        GAME_VERSION,
		Winner:         winner,
		HighlightGates: highlightGates,
	}

	dataStr, _ := json.Marshal(g.gameStateData)

	resultStr := string(winner)
	err = g.Server.SaveGameResult(g.GameNumber, g.GameId, g.RoundId, g.StateMng.CurrState, g.StateMng.StateTime, resultStr, string(dataStr), "", "")
	if err != nil {
		com.VUtils.PrintError(err)
		g.Server.Maintenance()
		return
	}

	// save new trend item
	trendItem := com.TrendItem{GameNumber: g.GameNumber, RoundId: g.RoundId, Result: resultStr, Data: string(dataStr), Txh: "", W: ""}
	g.Trends = append([]*com.TrendItem{&trendItem}, g.Trends...)
	if len(g.Trends) > com.TREND_PAGE_SIZE {
		g.Trends = g.Trends[:com.TREND_PAGE_SIZE]
	}

	for _, room := range g.RoomList {
		content := GenResultContent{Cock1: g.gameStateData.Cock_1, Cock2: g.gameStateData.Cock_2, Randoms: battleInfo.Randoms}
		res := (&com.ClientGenResultResponse{}).Init(room, content)
		room.BroadcastMessage(res)
	}

	fmt.Println("genResult battleInfo.Duration", battleInfo.Duration)
	// wait for the battle to play
	g.StateMng.SetStateDuration(com.GAME_STATE_GEN_RESULT, battleInfo.Duration)
}
