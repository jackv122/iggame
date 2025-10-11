package roul

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	com "vgame/_common"
)

type Roulette struct {
	com.BaseGame
	// payout
	GameData *RouletteData

	// Roulette-specific fields
	PathIds   []int
	PathInd   int
	ResultNum int
}

// work as constructor
func (g *Roulette) Init(server *com.GameServer) *Roulette {
	g.InitBase(server, com.IDRoulette, "Roulette")
	gameStates := []com.GameState{com.GAME_STATE_STARTING, com.GAME_STATE_BETTING, com.GAME_STATE_CLOSE_BETTING, com.GAME_STATE_GEN_RESULT, com.GAME_STATE_RESULT, com.GAME_STATE_PAYOUT}
	stateTimes := []float64{1, 30, 3, 0, 10, 8.0} // 0 mean wait forever
	g.StateMng = (&com.StateManager{}).Init(gameStates, stateTimes, g.onEnterState, g.onExitState)
	g.GameData = (&RouletteData{}).init(g)

	// Initialize Roulette-specific fields
	g.ResultNum = -1
	pathLength := 24
	for i := 0; i < pathLength; i++ {
		g.PathIds = append(g.PathIds, i)
	}
	g.ShuffleArr()

	// Limit maps are now loaded from JSON config in BaseGame

	return g
}

func (g *Roulette) Start() {
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

func (g *Roulette) ShuffleArr() {
	var t int = 0
	l := len(g.PathIds)
	for i := 0; i < l; i++ {
		ind := int(math.Floor(com.VUtils.GetRandFloat64() * (float64(l) - 0.01)))
		t = g.PathIds[i]
		g.PathIds[i] = g.PathIds[ind]
		g.PathIds[ind] = t
	}
}

func (g *Roulette) GetResultNum() int {
	return g.ResultNum
}

func (g *Roulette) GetResultData() interface{} {
	return fmt.Sprintf("%d_%d", g.ResultNum, g.PathIds[g.PathInd])
}

func (g *Roulette) OnStartComplete() {
	fmt.Println("Roulette start complete")
	g.StateMng.Start()
	gameConf := com.GameServerConfig.GameConfigMap[g.GameId]
	com.VUtils.RepeatCall(g.Update, gameConf.FrameTime, 0, g.GetTimeKeeper())
}

func (g *Roulette) LoadGameState() bool {
	row := g.Server.DB.QueryRow("SELECT gamenumber, roundid, state, statetime, result, tx, w, h FROM gamestate WHERE gameid=?", g.GameId)

	var currState = com.GameState(com.GAME_STATE_STARTING)
	result := ""
	hash := ""
	statetime := float64(0)
	err := row.Scan(&g.GameNumber, &g.RoundId, &currState, &statetime, &result, &g.Txh, &g.W, &hash)
	if err != nil {
		fmt.Println("not existing previous state data for game ", g.GameId)
		return false
	}
	str := fmt.Sprintf("%d_%s_%d_%d_%s_%s_%s", g.GameNumber, g.GameId, g.RoundId, currState, result, g.Txh, g.W)

	if hash != com.VUtils.HashString(str) {
		fmt.Println("Wrong hash for game ", g.GameId)
		g.Server.Stop()
		return false
	}
	if result != "" {
		g.ResultNum, err = strconv.Atoi(result)
		if err != nil {
			msg := fmt.Sprintf("can not parse game result for gameId %s and result %s ", g.GameId, result)
			com.VUtils.PrintError(errors.New(msg))
			g.Server.Maintenance()
			return true
		}
	}

	// resume state
	g.StateMng.SetState(currState, statetime)

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

// game results --------------------------------------------

func (g *Roulette) onEnterState(state com.GameState) {
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

func (g *Roulette) onExitState(state com.GameState) {
}

func (g *Roulette) OnEnterStarting() {
	fmt.Println("Roulette entering STARTING state")
	g.RoundId++
	g.PathInd++

	if g.PathInd > len(g.PathIds)-1 {
		g.PathInd = 0
		g.ShuffleArr()
	}

	g.Txh = ""
	g.W = ""
	g.TickTime = 0
	g.ResultNum = -1
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
	response, err3 := tx.Exec("INSERT INTO gamestate(gameid, roundid, state, statetime, result) VALUES(?,?,?,?,?)", g.GameId, g.RoundId, com.GAME_STATE_STARTING, 0, "")
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
		res := (&com.BaseGameResponse{}).Init(room, com.CMD_START_GAME)
		room.BroadcastMessage(res)
	}
}

func (g *Roulette) OnEnterBetting() {
	g.BaseGame.OnEnterBetting()
}

func (g *Roulette) OnEnterCloseBetting() {
	g.BaseGame.OnEnterCloseBetting()
}

func (g *Roulette) OnEnterGenResult() {
	fmt.Printf("%s entering GEN_RESULT state\n", g.Name)
	for _, room := range g.RoomList {
		room.NotifyEndBetting()
	}
	g.genResult()
}

func (g *Roulette) OnEnterResult() {
	fmt.Println("Roulette entering RESULT state")
	if g.ResultNum < 0 {
		msg := fmt.Sprintf("game %s has no result when payout", g.GameId)
		com.VUtils.PrintError(errors.New(msg))
		g.Server.Maintenance()
		return
	}
	// calculate payout & save DB but not send payout to player, payout should send when state payout start
	for _, room := range g.RoomList {
		result := strconv.Itoa(g.ResultNum) + "_" + strconv.Itoa(g.PathIds[g.PathInd])
		res := (&com.ClientGameResultResponse{}).Init(room, com.CMD_GAME_RESULT, result, g.Txh, g.W)
		room.BroadcastMessage(res)
	}
	// run it on other thread
	go g.payout()
}

func (g *Roulette) OnEnterPayout() {
	g.BaseGame.OnEnterPayout()
}

func (g *Roulette) payout() {
	if g.ResultNum < 0 {
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
	//fmt.Println("payout success for gamenumber", g.GameNumber)
}

func (g *Roulette) payoutRoom(room *com.GameRoom) bool {
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
			isWin, has := g.GameData.betResultMap[string(betPlace.Type)][g.ResultNum]
			if has && isWin {
				betKind := g.BetKindMap[string(betPlace.Type)]
				betPay = g.PayoutMap[string(betKind)] * betPlace.Amount
				totalPay += betPay
			}
			// apply payout also for not win bet
			confirmPayouts = append(confirmPayouts, &com.PayoutInfo{BetType: betPlace.Type, BetAmount: betPlace.Amount, PayoutAmount: betPay})
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
				success = false
			}
			//userId := res.UserId // alternative way
			betInfo := room.BetInfosMap[userId]
			betInfo.Balance = res.Balance
		})

	}
	return success
}

func (g *Roulette) genResult() {
	// if resume state
	if g.ResultNum >= 0 {
		fmt.Println("g.ResultNum existing.")
		g.StateMng.NextState()
		return
	}
	go func() {
		maxTry := 3
		for tryCount := 0; tryCount < maxTry; tryCount++ {
			fmt.Println("genresult tryCount ", tryCount)
			apiUrl := com.BLOCKCHAIN_URL + "?sender=a10&amount=" + fmt.Sprintf("%06d", g.GetRoundId())
			// create new http request
			response, err := http.Get(apiUrl)
			if err != nil {
				com.VUtils.PrintError(err)
				if tryCount < maxTry {
					continue
				} else {
					return
				}
			}

			defer func() {
				response.Body.Close()
			}()

			responseBody, err := io.ReadAll(response.Body)

			if err != nil {
				com.VUtils.PrintError(err)
				return
			}

			txResult := com.BlockChainTxResult{}
			txResult.ErrorMessage = ""
			err = json.Unmarshal(responseBody, &txResult)

			if err != nil {
				com.VUtils.PrintError(err)
				return
			}

			if txResult.ErrorCode != 0 {
				if tryCount < maxTry {
					//time.Sleep(time.Duration(2+com.VUtils.GetRandInt(2)) * time.Second)
					fmt.Println("txResult.ErrorCode: ", txResult.ErrorCode, " tryCount: ", tryCount)
					continue
				} else {
					com.VUtils.PrintError(errors.New("txResult.ErrorMessage " + txResult.ErrorMessage))
					return
				}
			}
			g.Txh = txResult.Txh
			g.W = txResult.W
			// success
			last := txResult.Txh[len(txResult.Txh)-2:]
			n := new(big.Int) // big endian
			n.SetString(last, 16)
			var v int = int(n.Int64())
			rand := float64(v) / float64(0xff)
			//fmt.Println("txh ", txResult.Txh, " rand ", formatAmount(Amount(rand)))
			//g.ResultNum = com.VUtils.GetRandInt(37)
			g.ResultNum = int(rand * float64(36+0.9999))
			resultStr := strconv.Itoa(g.ResultNum)
			err = g.Server.SaveGameResult(g.GameNumber, g.GameId, g.RoundId, g.StateMng.CurrState, g.StateMng.StateTime, resultStr, "", txResult.Txh, txResult.W)
			if err != nil {
				com.VUtils.PrintError(err)
				g.Server.Maintenance()
				return
			}
			g.StateMng.NextState()
			// save new trend item
			trendItem := com.TrendItem{GameNumber: g.GameNumber, RoundId: g.RoundId, Result: resultStr, Txh: txResult.Txh, W: txResult.W}
			g.Trends = append([]*com.TrendItem{&trendItem}, g.Trends...)
			if len(g.Trends) > com.TREND_PAGE_SIZE {
				g.Trends = g.Trends[:com.TREND_PAGE_SIZE]
			}

			// success, break for loop
			break
		}

	}()
}

// end state machine -------------------------------------------

func (g *Roulette) SaveGameState() {
	resultStr := strconv.Itoa(g.ResultNum)
	str := fmt.Sprintf("%d_%s_%d_%d_%s_%s_%s", g.GameNumber, g.GameId, g.RoundId, g.StateMng.CurrState, resultStr, g.Txh, g.W)
	hash := com.VUtils.HashString(str)
	_, err := g.Server.DB.Exec("UPDATE gamestate SET state=?, statetime=?, result=?, tx=?, w=?, h=? WHERE gamenumber=?", g.StateMng.CurrState, g.StateMng.StateTime, resultStr, g.Txh, g.W, hash, g.GameNumber)
	if err != nil {
		com.VUtils.PrintError(err)
		g.Server.Maintenance()
		return
	}
}

/*
BetRequest

	{
		public Cmd = GameCMD.SEND_BET_UPDATE;
		public RoomId:string;
		public RoundId:number;
		public BetTypes:Array<string>;
		public Amounts:Array<number>;
		public Time:string;
		public Platform:string;
	}
*/
func (game *Roulette) OnMessage(roomId com.RoomId, connId com.ConnectionId, msg string) {
	//fmt.Printf("game: %s OnMessage: %s\n", game.name, *msg)
	connInfo, ok := game.Server.GetConnectionInfo(connId)
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
	room := game.Server.RoomMng.GetRoom(connInfo.OperatorId, com.RoomId(roomId))
	if room == nil {
		fmt.Println("OnMessage roomId == nil ", cmd)
		return
	}
	currState := game.StateMng.CurrState
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
			game.Server.SendPrivateMessage(room.RoomId, connId, res)
		}
	}

	room.OnMessage(cmd, connInfo, msg)
}

func (game *Roulette) Stop() {
	if game.TimeKeeper.Timer != nil {
		game.TimeKeeper.Timer.Stop()
	}
}

func (game *Roulette) GetTimeKeeper() *com.TimeKeeper {
	return game.TimeKeeper
}

func (game *Roulette) GetRoundId() com.RoundId {
	return game.RoundId
}

func (game *Roulette) GetGameNumber() com.GameNumber {
	return game.GameNumber
}

func (game *Roulette) GetBetKind(betType com.BetType) com.BetKind {
	return game.BetKindMap[string(betType)]
}

func (game *Roulette) GetAllBetLimits() []map[com.Currency]map[com.BetKind]*com.BetLimit {
	limits := []map[com.Currency]map[com.BetKind]*com.BetLimit{game.SmallLimitBetMap, game.MediumLimitBetMap}
	return limits
}

func (game *Roulette) GetBetLimit(level com.LimitLevel) map[com.Currency]map[com.BetKind]*com.BetLimit {
	switch level {
	case com.LIMIT_LEVEL_SMALL:
		return game.SmallLimitBetMap
	case com.LIMIT_LEVEL_MEDIUM:
		return game.MediumLimitBetMap
	}

	return game.SmallLimitBetMap
}

func (game *Roulette) GetPayout(betKind com.BetKind) com.Amount {
	return game.PayoutMap[string(betKind)]
}

func (game *Roulette) GetGameResultString() string {
	if game.ResultNum < 0 {
		return ""
	}
	return strconv.Itoa(game.ResultNum)
}
