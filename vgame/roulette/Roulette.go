package rou

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	com "vgame/_common"
)

type Roulette struct {
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
	GameData   *RouletteData
	gameNumber com.GameNumber // need to gen on state STARTING

	SmallLimitBetMap  map[com.Currency]map[com.BetKind]*com.BetLimit
	MediumLimitBetMap map[com.Currency]map[com.BetKind]*com.BetLimit
	pathIds           []int
	pathInd           int
	trends            []*com.TrendItem

	tickCount int
}

// work as constructor
func (g *Roulette) Init(server *com.GameServer) *Roulette {
	g.server = server
	g.gameId = com.IDRoulette
	g.name = "Roulette"
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
	g.GameData = (&RouletteData{}).init()

	// Load limit maps from JSON configuration
	g.loadLimitMapsFromJSON()
	return g
}

func (g *Roulette) suffleArr() {
	var t int = 0
	l := len(g.pathIds)
	for i := 0; i < l; i++ {
		ind := int(math.Floor(com.VUtils.GetRandFloat64() * (float64(l) - 0.01)))
		t = g.pathIds[i]
		g.pathIds[i] = g.pathIds[ind]
		g.pathIds[ind] = t
	}
}

func (g *Roulette) loadLimitMapsFromJSON() {
	configPath := "config/roulette_config.json"

	// Check if config file exists, if not use default values
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Roulette config file %s not found, using default values\n", configPath)
		g.loadDefaultLimitMaps()
		return
	}

	// Read and parse JSON config
	jsonData, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading roulette config file: %v, using default values\n", err)
		g.loadDefaultLimitMaps()
		return
	}

	var configData map[string]interface{}
	err = json.Unmarshal(jsonData, &configData)
	if err != nil {
		fmt.Printf("Error parsing roulette config file: %v, using default values\n", err)
		g.loadDefaultLimitMaps()
		return
	}

	// Initialize maps
	g.SmallLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	g.MediumLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}

	// Load small limit bet map
	if smallLimitData, ok := configData["smallLimitBetMap"].(map[string]interface{}); ok {
		for currency, currencyData := range smallLimitData {
			if currencyMap, ok := currencyData.(map[string]interface{}); ok {
				betMap := map[com.BetKind]*com.BetLimit{}
				for betKindStr, limitData := range currencyMap {
					if betKindInt, err := strconv.Atoi(betKindStr); err == nil {
						if limitMap, ok := limitData.(map[string]interface{}); ok {
							if minVal, ok := limitMap["min"].(float64); ok {
								if maxVal, ok := limitMap["max"].(float64); ok {
									betMap[com.BetKind(betKindInt)] = &com.BetLimit{
										Min: com.Amount(minVal),
										Max: com.Amount(maxVal),
									}
								}
							}
						}
					}
				}
				g.SmallLimitBetMap[com.Currency(currency)] = betMap
			}
		}
	}

	// Load medium limit bet map
	if mediumLimitData, ok := configData["mediumLimitBetMap"].(map[string]interface{}); ok {
		for currency, currencyData := range mediumLimitData {
			if currencyMap, ok := currencyData.(map[string]interface{}); ok {
				betMap := map[com.BetKind]*com.BetLimit{}
				for betKindStr, limitData := range currencyMap {
					if betKindInt, err := strconv.Atoi(betKindStr); err == nil {
						if limitMap, ok := limitData.(map[string]interface{}); ok {
							if minVal, ok := limitMap["min"].(float64); ok {
								if maxVal, ok := limitMap["max"].(float64); ok {
									betMap[com.BetKind(betKindInt)] = &com.BetLimit{
										Min: com.Amount(minVal),
										Max: com.Amount(maxVal),
									}
								}
							}
						}
					}
				}
				g.MediumLimitBetMap[com.Currency(currency)] = betMap
			}
		}
	}

	fmt.Println("Roulette configuration loaded from JSON file")
}

func (g *Roulette) loadDefaultLimitMaps() {
	g.SmallLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	usdc := map[com.BetKind]*com.BetLimit{}

	usdc[BET_Straight] = &com.BetLimit{Min: 0.1, Max: 3}
	usdc[BET_Split] = &com.BetLimit{Min: 0.1, Max: 6}
	usdc[BET_Street] = &com.BetLimit{Min: 0.1, Max: 9}
	usdc[BET_Corner] = &com.BetLimit{Min: 0.1, Max: 12}
	usdc[BET_Line] = &com.BetLimit{Min: 0.1, Max: 18}
	usdc[BET_Trio] = &com.BetLimit{Min: 0.1, Max: 9}
	usdc[BET_Basket] = &com.BetLimit{Min: 0.1, Max: 16}
	usdc[BET_Odd_Even] = &com.BetLimit{Min: 0.1, Max: 54}
	usdc[BET_Red_Black] = &com.BetLimit{Min: 0.1, Max: 54}
	usdc[BET_High_Low] = &com.BetLimit{Min: 0.1, Max: 54}
	usdc[BET_Columns] = &com.BetLimit{Min: 0.1, Max: 36}
	usdc[BET_Dozens] = &com.BetLimit{Min: 0.1, Max: 36}
	g.SmallLimitBetMap[com.Currency("USDC")] = usdc

	g.MediumLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	usdc = map[com.BetKind]*com.BetLimit{}
	usdc[BET_Straight] = &com.BetLimit{Min: 1, Max: 30}
	usdc[BET_Split] = &com.BetLimit{Min: 1, Max: 60}
	usdc[BET_Street] = &com.BetLimit{Min: 1, Max: 90}
	usdc[BET_Corner] = &com.BetLimit{Min: 1, Max: 120}
	usdc[BET_Line] = &com.BetLimit{Min: 1, Max: 180}
	usdc[BET_Trio] = &com.BetLimit{Min: 1, Max: 90}
	usdc[BET_Basket] = &com.BetLimit{Min: 1, Max: 160}
	usdc[BET_Odd_Even] = &com.BetLimit{Min: 1, Max: 540}
	usdc[BET_Red_Black] = &com.BetLimit{Min: 1, Max: 540}
	usdc[BET_High_Low] = &com.BetLimit{Min: 1, Max: 540}
	usdc[BET_Columns] = &com.BetLimit{Min: 1, Max: 360}
	usdc[BET_Dozens] = &com.BetLimit{Min: 1, Max: 360}
	g.MediumLimitBetMap[com.Currency("USDC")] = usdc
}

func (g *Roulette) Start() {
	fmt.Println("Roulette start")
	trends := g.server.LoadTrends(g.gameId, 0)
	if trends != nil {
		g.trends = trends
	}
	if !g.LoadGameState() {
		g.stateMng.ResetState()
		g.onStartComplete()
	}
}

func (g *Roulette) onStartComplete() {
	fmt.Println("Roulette start complete")
	g.stateMng.Start()
	gameConf := com.GameServerConfig.GameConfigMap[g.gameId]
	com.VUtils.RepeatCall(g.Update, gameConf.FrameTime, 0, g.GetTimeKeeper())
}

func (g *Roulette) LoadGameState() bool {
	row := g.server.DB.QueryRow("SELECT gamenumber, roundid, state, statetime, result, tx, w, h FROM gamestate WHERE gameid=?", g.gameId)

	var currState = com.GameState(com.GAME_STATE_STARTING)
	result := ""
	hash := ""
	statetime := float64(0)
	err := row.Scan(&g.gameNumber, &g.roundId, &currState, &statetime, &result, &g.txh, &g.w, &hash)
	if err != nil {
		fmt.Println("not existing previous state data for game ", g.gameId)
		return false
	}
	str := fmt.Sprintf("%d_%s_%d_%d_%s_%s_%s", g.gameNumber, g.gameId, g.roundId, currState, result, g.txh, g.w)

	if hash != com.VUtils.HashString(str) {
		fmt.Println("Wrong hash for game ", g.gameId)
		g.server.Stop()
		return false
	}
	if result != "" {
		g.resultNum, err = strconv.Atoi(result)
		if err != nil {
			msg := fmt.Sprintf("can not parse game result for gameId %s and result %s ", g.gameId, result)
			com.VUtils.PrintError(errors.New(msg))
			g.server.Maintenance()
			return true
		}
	}

	// resume state
	g.stateMng.SetState(currState, statetime)

	// reload all bettings
	gameConf := com.GameServerConfig.GameConfigMap[g.gameId]
	loadRequestCount := len(gameConf.OperatorIds)
	for _, operatorId := range gameConf.OperatorIds {
		param := com.VUtils.WalletLocalMessageUint64(operatorId, com.WCMD_QUERY_BETTING, uint64(g.gameNumber))
		g.server.WalletConn.Send(param, func(vs *com.VSocket, requestId uint64, resData []byte) {
			res := com.QueryBettingResponse{}
			err := json.Unmarshal(resData, &res)
			if err != nil {
				com.VUtils.PrintError(err)
				g.server.Maintenance()
				return
			}
			for _, roomConf := range gameConf.RoomConfigs {
				room := g.server.RoomMng.GetRoom(operatorId, roomConf.RoomId)
				room.ResumeBetting(res.Bettings)
			}
			fmt.Println("loadRequestCount ===")
			loadRequestCount--
			if loadRequestCount == 0 {
				g.onStartComplete()
			}
		}, nil)
	}

	return true
}

// game results --------------------------------------------

func (g *Roulette) onTick() {
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

func (g *Roulette) onEnterState(state com.GameState) {
	// only broadcast for the users joined room already, so use room to broadcast insteak
	fmt.Println("onEnterState ", state)
	switch state {
	case com.GAME_STATE_STARTING:
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
	case com.GAME_STATE_BETTING:
		for _, room := range g.roomList {
			res := (&com.BaseGameResponse{}).Init(room, com.CMD_START_BET_SUCCEED)
			room.BroadcastMessage(res)
		}
	case com.GAME_STATE_CLOSE_BETTING:
		for _, room := range g.roomList {
			res := (&com.BaseGameResponse{}).Init(room, com.CMD_STOP_BET_SUCCEED)
			room.BroadcastMessage(res)
		}
	case com.GAME_STATE_GEN_RESULT:
		for _, room := range g.roomList {
			room.NotifyEndBetting()
		}
		g.genResult()
	case com.GAME_STATE_RESULT:
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
	case com.GAME_STATE_PAYOUT:
		// broadcast payout to users ---
		for _, room := range g.roomList {
			room.NotifyReward()
		}
	}
	g.SaveGameState()
}

func (g *Roulette) payout() {
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

func (g *Roulette) genResult() {
	// if resume state
	if g.resultNum >= 0 {
		fmt.Println("g.resultNum existing.")
		g.stateMng.NextState()
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
			g.txh = txResult.Txh
			g.w = txResult.W
			// success
			last := txResult.Txh[len(txResult.Txh)-2:]
			n := new(big.Int) // big endian
			n.SetString(last, 16)
			var v int = int(n.Int64())
			rand := float64(v) / float64(0xff)
			//fmt.Println("txh ", txResult.Txh, " rand ", formatAmount(Amount(rand)))
			//g.resultNum = com.VUtils.GetRandInt(37)
			g.resultNum = int(rand * float64(36+0.9999))
			resultStr := strconv.Itoa(g.resultNum)
			err = g.server.SaveGameResult(g.gameNumber, g.gameId, g.roundId, g.stateMng.CurrState, g.stateMng.StateTime, resultStr, txResult.Txh, txResult.W)
			if err != nil {
				com.VUtils.PrintError(err)
				g.server.Maintenance()
				return
			}
			g.stateMng.NextState()
			// save new trend item
			trendItem := com.TrendItem{GameNumber: g.gameNumber, RoundId: g.roundId, Result: resultStr, Txh: txResult.Txh, W: txResult.W}
			g.trends = append([]*com.TrendItem{&trendItem}, g.trends...)
			if len(g.trends) > com.TREND_PAGE_SIZE {
				g.trends = g.trends[:com.TREND_PAGE_SIZE]
			}

			// success, break for loop
			break
		}

	}()
}

func (g *Roulette) onExitState(state com.GameState) {

}

// end state machine -------------------------------------------

func (g *Roulette) InitRoomForGame(room *com.GameRoom) {
	g.roomList = append(g.roomList, room)
	// init limit map

}

func (g *Roulette) SaveGameState() {
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

// run on 1 owner timer thread
func (game *Roulette) Update(dt float64) {
	game.stateMng.StateUpdate(dt)
	game.tickTime += dt
	if game.tickTime > 1.0 {
		game.onTick()
		game.tickTime -= 1.0
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
func (game *Roulette) OnMessage(connId com.ConnectionId, msg string) {
	//fmt.Printf("game: %s OnMessage: %s\n", game.name, *msg)
	connInfo, ok := game.server.GetConnectionInfo(connId)
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
	room := game.server.RoomMng.GetRoom(connInfo.OperatorId, com.RoomId(roomId))
	if room == nil {
		fmt.Println("OnMessage roomId == nil ", cmd)
		return
	}
	currState := game.stateMng.CurrState
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
			game.server.SendPrivateMessage(connId, res)
		}
	}

	room.OnMessage(cmd, connInfo, msg)
}

func (game *Roulette) Stop() {
	if game.timeKeeper.Timer != nil {
		game.timeKeeper.Timer.Stop()
	}
}

func (game *Roulette) GetTimeKeeper() *com.TimeKeeper {
	return game.timeKeeper
}

func (game *Roulette) GetRoundId() com.RoundId {
	return game.roundId
}

func (game *Roulette) GetGameNumber() com.GameNumber {
	return game.gameNumber
}

func (game *Roulette) GetBetKind(betType com.BetType) com.BetKind {
	return game.GameData.BetKindMap[string(betType)]
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

func (game *Roulette) GetCurState() com.GameState {
	return game.stateMng.CurrState
}

func (game *Roulette) GetRemainStateTime() float64 {
	return game.stateMng.StateDurs[game.stateMng.CurrState] - game.stateMng.StateTime
}

func (game *Roulette) GetTotalBetTime() float64 {
	return game.stateMng.StateDurs[com.GAME_STATE_BETTING]
}

func (game *Roulette) GetGameResult() string {
	if game.resultNum < 0 {
		return ""
	}
	return strconv.Itoa(game.resultNum)
}

func (game *Roulette) LoadTrends(gameId com.GameId, page uint32) []*com.TrendItem {
	return game.trends
}

func (game *Roulette) GetTxh() string {
	return game.txh
}
func (game *Roulette) GetW() string {
	return game.w
}

func (game *Roulette) GetResultString() string {
	return strconv.Itoa(game.resultNum) + "_" + strconv.Itoa(game.pathIds[game.pathInd])
}
