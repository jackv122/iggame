package com

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type BetLimit struct {
	Min Amount
	Max Amount
}

type BetPlace struct {
	Type   BetType
	Amount Amount
}

type PayoutInfo struct {
	BetType      BetType
	BetAmount    Amount
	PayoutAmount Amount
}

type RoomConfig struct {
	RoomId      RoomId                             `json:"roomId"`
	LimitLevel  LimitLevel                         `json:"limitLevel"`
	limitBetMap map[Currency]map[BetKind]*BetLimit `json:"-"`
}

type UserBetInfo struct {
	UserId      UserId
	DbBettingId BettingId
	TotalPay    Amount
	Balance     Amount
	Currency    Currency
	Payedout    uint8
	Mutex       sync.Mutex

	ConfirmedBetState []*BetPlace
	ConfirmedPayouts  []*PayoutInfo
	SendingBetState   []*BetPlace
	WaitingBetState   []*BetPlace
}

func (u *UserBetInfo) init(userId UserId) *UserBetInfo {
	u.UserId = userId

	u.ConfirmedBetState = []*BetPlace{}
	u.SendingBetState = nil
	u.WaitingBetState = nil
	return u
}

func (u *UserBetInfo) resetBet() {
	//fmt.Println("reset bet for user ==== ", u.UserId)
	u.Mutex.Lock()
	defer u.Mutex.Unlock()
	u.ConfirmedBetState = []*BetPlace{}
	u.SendingBetState = nil
	u.WaitingBetState = nil
	u.Payedout = 0
	u.TotalPay = 0
	u.DbBettingId = 0
}

func (u *UserBetInfo) getSendingBetState() []*BetPlace {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()
	state := u.SendingBetState
	return state
}

func (u *UserBetInfo) setWatingBetState(state []*BetPlace) {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()
	u.WaitingBetState = state
}

type GameRoom struct {
	operatorId OperatorID
	Server     *GameServer
	RoomId     RoomId
	GameId     GameId

	time     float64
	connList []ConnectionId
	maxConn  int

	betTypeMap map[BetType]bool

	// map UserId to betlist
	BetInfosMap map[UserId]*UserBetInfo

	roomConfig *RoomConfig

	roomStatsChanged bool

	GameInitData interface{}
}

func (room *GameRoom) Init(server *GameServer, operatorId OperatorID) *GameRoom {
	room.operatorId = operatorId
	room.Server = server
	room.time = 0
	room.maxConn = 10000
	room.roomStatsChanged = true
	room.connList = []ConnectionId{}
	room.GameInitData = nil
	room.BetInfosMap = map[UserId]*UserBetInfo{}

	// checking valid bettype for better debugging
	room.betTypeMap = map[BetType]bool{}

	return room
}

func (room *GameRoom) initBetTypeList(types []BetType) {
	room.betTypeMap = map[BetType]bool{}
	for _, betType := range types {
		room.betTypeMap[betType] = true
	}
}

func (room *GameRoom) checkUserExist(userId UserId) (bool, ConnectionId) {
	for _, connId := range room.connList {
		conInfo, ok := room.Server.GetConnectionInfo(connId)
		if !ok { // it is a hot_fixed
			continue
		}
		if conInfo.UserId == userId {
			return true, connId
		}
	}
	return false, 0
}

func (room *GameRoom) JoinRoom(connId ConnectionId) bool {
	connExisting := false
	for _, connid := range room.connList {
		if connid == connId {
			connExisting = true
			break
		}
	}

	if !connExisting {
		room.connList = append(room.connList, connId)
		room.roomStatsChanged = true
		connInfo, ok := room.Server.GetConnectionInfo(connId)
		if !ok {
			return false
		}
		connInfo.joinRoom(room.RoomId)
		//fmt.Println("JoinRoom: ", room.RoomId, room.connList)
		betInfo, has := room.BetInfosMap[connInfo.UserId]

		if !has {
			room.BetInfosMap[connInfo.UserId] = (&UserBetInfo{}).init(connInfo.UserId)
			betInfo = room.BetInfosMap[connInfo.UserId]
		}
		// query balance
		room.getbalance(connInfo.UserId, func(balanceInfo *BalanceInfo) {
			betInfo.Balance = balanceInfo.Amount
			//fmt.Println("betInfo.Balance === ", betInfo.Balance)
			betInfo.Currency = balanceInfo.Currency

			res := (&ClientJoinRoomRes{}).Init(room, connInfo.UserId, betInfo.Balance)
			room.Server.SendPrivateMessage(room.RoomId, connId, res)
		})
		return true
	}

	return false
}

func (room *GameRoom) LeaveRoom(connId ConnectionId) {
	connInfo, ok := room.Server.GetConnectionInfo(connId)
	//fmt.Println("Leave Room === ", connInfo.UserId)
	if ok {
		connInfo.leaveRoom(room.RoomId)
	}
	room.connList = RemoveElementFromArray(room.connList, connId)
	room.roomStatsChanged = true
}

// send to all acive users
func (room *GameRoom) NotifyReward() {
	if len(room.connList) == 0 {
		return
	}
	for _, connId := range room.connList {
		connInfo, ok := room.Server.GetConnectionInfo(connId)
		if !ok {
			continue
		}
		playerPayouts := []*PlayerPayout{}
		res := (&ClientPayoutResponse{}).Init(room, &playerPayouts, 0)

		betInfo, has := room.BetInfosMap[connInfo.UserId]
		if has && betInfo.Payedout == 1 {

			payouts := []*PayoutInfo{}
			for _, payout := range betInfo.ConfirmedPayouts {
				payouts = append(payouts, payout)
			}
			playerPayout := PlayerPayout{Payouts: &payouts}

			res.PayoutContent.PlayerPayouts = &[]*PlayerPayout{&playerPayout}
			res.PayoutContent.Balance = truncateAmount(betInfo.Balance)
		}

		room.sendMessage(connId, res)
	}
}

func (room *GameRoom) NotifyRoomStats() {
	if !room.roomStatsChanged {
		return
	}
	room.roomStatsChanged = false
	res := (&ClientRoomStatsResponse{}).Init(room, CMD_LIVE_BET_STATS)
	res.UserCount = len(room.connList)
	res.TotalBet = room.GetRoomTotalBet()
	room.BroadcastMessage(res)
}

func (room *GameRoom) getbalances(ul []UserId, completeHdl func(balanceInfos []*BalanceInfo)) {
	bytes := VUtils.Uint16ToBytes(WCMD_GET_BALANCE_LIST)
	body, _ := json.Marshal(ul)
	bytes = append(bytes, body...)

	ops := []byte(room.operatorId)
	bytes = append(ops, bytes...)

	room.Server.WalletConn.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
		response := BalanceListResponse{}
		err := json.Unmarshal(resData, &response)
		if err != nil {
			panic("response getBalances fail")
		}
		completeHdl(response.BalanceInfos)
	}, nil)
}

func (room *GameRoom) getbalance(userId UserId, completeHdl func(balanceInfo *BalanceInfo)) {
	bytes := VUtils.WalletLocalMessageUint64(room.operatorId, WCMD_GET_BALANCE, uint64(userId))

	room.Server.WalletConn.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
		response := BalanceResponse{}
		err := json.Unmarshal(resData, &response)
		if err != nil {
			panic("response getBalances fail")
		}
		completeHdl(response.BalanceInfo)
	}, nil)
}

// send to all users in room
func (room *GameRoom) NotifyEndBetting() {
	if len(room.connList) == 0 {
		return
	}
	for _, connId := range room.connList {
		connInfo, ok := room.Server.GetConnectionInfo(connId)
		if !ok {
			continue
		}
		betInfo, has := room.BetInfosMap[connInfo.UserId]
		if has {
			res := (&EndBetResponse{}).Init(room)
			res.BetState = betInfo.ConfirmedBetState
			res.Balance = truncateAmount(betInfo.Balance)
			room.sendMessage(connId, res)
		}
	}
}

func (room *GameRoom) sendMessage(connId ConnectionId, data any) {
	room.Server.SendPrivateMessage(room.RoomId, connId, data)
}

// send to all acive users
func (room *GameRoom) BroadcastMessage(data any) {
	if len(room.connList) == 0 {
		return
	}
	room.Server.SendPublicMessage(room.RoomId, room.connList, data)
}

// save the current betting to DB
func (room *GameRoom) saveBets() {
}

// load the current betting from DB
func (room *GameRoom) loadBets() {
}

func (room *GameRoom) GetRoomTotalBet() Amount {
	roomTotalBet := Amount(0)
	for _, betInfo := range room.BetInfosMap {
		roomTotalBet += GetTotalBet(betInfo.ConfirmedBetState)
	}
	return roomTotalBet
}

func (room *GameRoom) ResetBets() {
	room.roomStatsChanged = true
	// reset bets
	leaveUsers := []UserId{}
	for userId, userBetInfo := range room.BetInfosMap {
		userBetInfo.resetBet()
		existing, _ := room.checkUserExist(userId)
		if !existing {
			leaveUsers = append(leaveUsers, userId)
		}
	}

	// optimize memory usage - ONLY can delete userBetInfo when start game
	for _, userId := range leaveUsers {
		delete(room.BetInfosMap, userId)
	}
}

// without payout
func getPureBetDetail(betState []*BetPlace) string {
	betDetail := ""
	for _, betPlace := range betState {
		if betDetail != "" {
			betDetail += ","
		}
		betDetail += string(betPlace.Type) + "_" + FormatAmount(betPlace.Amount) + "_0"
	}
	return betDetail
}

func GetTotalBet(betState []*BetPlace) Amount {
	var total Amount = 0
	for _, betPlace := range betState {
		total += betPlace.Amount
	}
	return total
}

func (room *GameRoom) OnMessage(cmd string, connInfo *ConnectionInfo, msg string) {
	//fmt.Println("OnMessage === ", cmd)
	switch cmd {
	case CMD_GET_ROOM_INFO:
		res := (&ClientRoomInfoResponse{}).Init(room, connInfo.UserId, room.GameInitData)
		room.Server.SendPrivateMessage(room.RoomId, connInfo.ConnId, res)
	case CMD_GET_TRENDS:
		game := GetGameInterface(room.GameId, room.Server)
		trends := game.LoadTrends(room.GameId, 0)
		if trends != nil {
			res := (&ClientTrendResponse{}).Init(room, &trends)
			room.Server.SendPrivateMessage(room.RoomId, connInfo.ConnId, res)
		}

	case CMD_LEAVE_ROOM:
		res := (&ClientNumberGameResponse{}).Init(room, CMD_LEAVE_ROOM_SUCCESS)
		room.LeaveRoom(connInfo.ConnId)
		room.Server.SendPrivateMessage(room.RoomId, connInfo.ConnId, res)
	}
}

func (room *GameRoom) ProcessBets(connId ConnectionId, clientRequestId string, state []*BetPlace) {
	betState := []*BetPlace{}

	// IMPORTANT: check valid betState ----
	// client should check bet min/max before sending ---
	msg := ""
	conInfo, ok := room.Server.GetConnectionInfo(connId)
	if !ok {
		return
	}
	betLimitMap := room.roomConfig.limitBetMap[conInfo.Currency]
	game := GetGameInterface(room.GameId, room.Server)
	for _, betPlace := range state {

		if len(room.betTypeMap) > 0 {
			_, has := room.betTypeMap[betPlace.Type]
			if !has {
				msg = fmt.Sprintf("gameId %s invalid bettype %s", room.GameId, betPlace.Type)
				VUtils.PrintError(errors.New(msg))
				return
			}
		}

		betKind := game.GetBetKind(betPlace.Type)

		if betPlace.Amount > 0 && betPlace.Amount < betLimitMap[betKind].Min {
			//msg = fmt.Sprintf("gameId %s invalid bet amount %f", room.GameId, betPlace.Amount)
			if true {
				res := (&BetFailResponse{}).Init(room)
				res.FailCode = RES_FAIL_BET_MIN
				room.sendMessage(connId, res)
			}
			continue
		}
		if betPlace.Amount > betLimitMap[betKind].Max {
			//msg = fmt.Sprintf("gameId %s invalid bet amount %f", room.GameId, betPlace.Amount)
			//fmt.Println(msg)
			if true {
				res := (&BetFailResponse{}).Init(room)
				res.FailCode = RES_FAIL_BET_MAX
				room.sendMessage(connId, res)
			}
			continue
		}

		// valid
		betState = append(betState, betPlace)
	}
	// ------------------------------------
	connInfo, has := room.Server.GetConnectionInfo(connId)
	if !has {
		VUtils.PrintError(errors.New("connection not existing " + strconv.Itoa(int(connId))))
		return
	}
	room.roomStatsChanged = true
	betInfo := room.BetInfosMap[connInfo.UserId]

	betInfo.setWatingBetState(betState)
	if betInfo.getSendingBetState() == nil {
		room.doBetting(clientRequestId, connInfo, betInfo)
	}
}

func (room *GameRoom) SaveBetting(betInfo *UserBetInfo, betDetail string, changeAmount Amount, completHdl func(vs *VSocket, requestId uint64, resData []byte)) {
	game := GetGameInterface(room.GameId, room.Server)

	// save betting ---

	bytes := VUtils.Uint16ToBytes(WCMD_SAVE_BETTING)
	param := BettingRecord{}

	param.Result = game.GetGameResultString()
	param.BettingId = betInfo.DbBettingId
	param.GameId = room.GameId
	param.GameNumber = game.GetGameNumber()
	param.RoundId = game.GetRoundId()
	param.RoomId = room.RoomId
	param.UserId = betInfo.UserId
	param.BetDetail = betDetail
	param.Payout = betInfo.TotalPay
	param.Payedout = betInfo.Payedout  // 0 or 1
	param.BalanceChange = changeAmount // it can be the change amount when (bet/undo bet/payout)

	body, _ := json.Marshal(param)
	bytes = append(bytes, body...)

	ops := []byte(room.operatorId)
	bytes = append(ops, bytes...)

	room.Server.WalletConn.Send(bytes, completHdl, nil)
}

func (room *GameRoom) ResumeBetting(bettings []*BettingRecord) {
	for _, betRecord := range bettings {
		game := GetGameInterface(room.GameId, room.Server)
		if betRecord.GameNumber == game.GetGameNumber() && betRecord.RoomId == room.RoomId {
			betInfo, has := room.BetInfosMap[betRecord.UserId]
			if !has {
				room.BetInfosMap[betRecord.UserId] = (&UserBetInfo{}).init(betRecord.UserId)
				betInfo = room.BetInfosMap[betRecord.UserId]
			}
			betInfo.Payedout = betRecord.Payedout
			betInfo.TotalPay = betRecord.Payout
			betInfo.DbBettingId = betRecord.BettingId
			betInfo.ConfirmedPayouts = []*PayoutInfo{}
			if betRecord.BetDetail != "" {
				arr := strings.Split(betRecord.BetDetail, ",")
				for _, s := range arr {
					payout := PayoutInfo{}
					arr2 := strings.Split(s, "_")
					payout.BetType = BetType(arr2[0])
					amt, _ := strconv.ParseFloat(arr2[1], 64)
					payout.BetAmount = Amount(amt)
					amt2, _ := strconv.ParseFloat(arr2[2], 64)
					payout.PayoutAmount = Amount(amt2)
					betInfo.ConfirmedPayouts = append(betInfo.ConfirmedPayouts, &payout)
				}
			}

			if betRecord.BetDetail != "" {
				arr := strings.Split(betRecord.BetDetail, ",")
				for _, s := range arr {
					betPlace := BetPlace{}
					arr2 := strings.Split(s, "_")
					betPlace.Type = BetType(arr2[0])
					amt, _ := strconv.ParseFloat(arr2[1], 64)
					betPlace.Amount = Amount(amt)
					betInfo.ConfirmedBetState = append(betInfo.ConfirmedBetState, &betPlace)
				}
			}
		}
	}
}

func (room *GameRoom) doBetting(clientRequestId string, connInfo *ConnectionInfo, betInfo *UserBetInfo) {

	betInfo.Mutex.Lock()
	if betInfo.WaitingBetState == nil {
		betInfo.Mutex.Unlock()
		return
	}
	betState := betInfo.WaitingBetState
	betInfo.WaitingBetState = nil
	betDetail := getPureBetDetail(betState)
	confirmedBet := GetTotalBet(betInfo.ConfirmedBetState)
	currentBet := GetTotalBet(betState)
	changeAmount := confirmedBet - currentBet

	// register sendingBetState until got wallet confirm
	betInfo.SendingBetState = betState
	// -------------------------------------------------
	betInfo.Mutex.Unlock()

	room.SaveBetting(betInfo, betDetail, changeAmount, func(vs *VSocket, requestId uint64, resData []byte) {
		res := BettingResponse{}
		err := json.Unmarshal(resData, &res)
		if err != nil {
			panic("response update balance fail")
		}

		betInfo.Mutex.Lock()

		// update dbBettingId
		betInfo.DbBettingId = res.BettingId

		if res.ErrorCode == 0 {
			betInfo.ConfirmedBetState = betInfo.SendingBetState
			// must update user betInfo balance here
			betInfo.Balance = res.Balance
			// success, no need to return new balance to client
			clientRes := (&ClientNumberGameResponse{}).Init(room, CMD_BET_UPDATE_SUCCEED)
			clientRes.ClientRequestId = clientRequestId
			clientRes.Val = truncateAmount(res.Balance)
			room.sendMessage(connInfo.ConnId, clientRes)
		} else {
			clientRes := (&BetFailResponse{}).Init(room)
			clientRes.ClientRequestId = clientRequestId

			clientRes.FailCode = res.ErrorCode
			room.sendMessage(connInfo.ConnId, clientRes)
		}

		// reset sendingBetState
		betInfo.SendingBetState = nil

		betInfo.Mutex.Unlock()

		// continue to check waitingBetState for sending saveBetting() again. The waitingBetState maybe changed in time of saveBetting process
		room.doBetting(clientRequestId, connInfo, betInfo)

	})
}

// betting logics
func (room *GameRoom) onStopBet() {
}
