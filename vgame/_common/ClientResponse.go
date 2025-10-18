package com

import (
	"time"
)

// BaseGameResponse -----------

type ClientRoomStatsResponse struct {
	CMD       string
	GameId    GameId
	RoomId    RoomId
	RoundId   RoundId
	UserCount int
	TotalBet  Amount
}

func (res *ClientRoomStatsResponse) Init(room *GameRoom, cmd string) *ClientRoomStatsResponse {
	res.CMD = cmd

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	res.UserCount = 0
	res.TotalBet = 0
	game := GetGameInterface(room.GameId, room.Server)
	res.RoundId = game.GetRoundId()
	return res
}

type ClientIntGameResponse struct {
	CMD     string
	GameId  GameId
	RoomId  RoomId
	RoundId RoundId
	IntVal  int
}

func (res *ClientIntGameResponse) Init(room *GameRoom, cmd string) *ClientIntGameResponse {
	res.CMD = cmd

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.RoundId = game.GetRoundId()
	return res
}

type ClientNumberGameResponse struct {
	CMD             string
	GameId          GameId
	ClientRequestId string
	RoomId          RoomId
	RoundId         RoundId
	Val             Amount
}

func (res *ClientNumberGameResponse) Init(room *GameRoom, cmd string) *ClientNumberGameResponse {
	res.CMD = cmd

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.RoundId = game.GetRoundId()
	return res
}

type ClientStringGameResponse struct {
	CMD        string
	GameId     GameId
	RoomId     RoomId
	GameNumber GameNumber
	RoundId    RoundId
	Str        string
}

func (res *ClientStringGameResponse) Init(room *GameRoom, cmd string, str string) *ClientStringGameResponse {
	res.CMD = cmd
	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.GameNumber = game.GetGameNumber()
	res.RoundId = game.GetRoundId()
	res.Str = str
	return res
}

type ClientGameResultResponse struct {
	CMD        string
	GameId     GameId
	RoomId     RoomId
	GameNumber GameNumber
	RoundId    RoundId
	Result     interface{}
	Txh        string
	W          string
}

func (res *ClientGameResultResponse) Init(room *GameRoom, cmd string, result interface{}, Txh string, W string) *ClientGameResultResponse {
	res.CMD = cmd
	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.GameNumber = game.GetGameNumber()
	res.RoundId = game.GetRoundId()
	res.Result = result
	res.Txh = Txh
	res.W = W
	return res
}

type ClientGenResultResponse struct {
	CMD        string
	GameId     GameId
	RoomId     RoomId
	GameNumber GameNumber
	RoundId    RoundId
	Content    interface{}
	Txh        string
	W          string
}

func (res *ClientGenResultResponse) Init(room *GameRoom, content interface{}) *ClientGenResultResponse {
	res.CMD = CMD_GEN_RESULT
	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.GameNumber = game.GetGameNumber()
	res.RoundId = game.GetRoundId()
	res.Content = content
	return res
}

type ClientJoinGameRes struct {
	CMD    string
	GameId GameId
}

func (res *ClientJoinGameRes) Init(gameId GameId) *ClientJoinGameRes {
	res.CMD = CMD_JOIN_ROOM_SUCCESS
	res.GameId = gameId
	return res
}

type ClientJoinRoomRes struct {
	CMD      string
	RoomId   RoomId
	GameId   GameId
	Balance  Amount
	RoomInfo *RoomInfoContent
}

func (res *ClientJoinRoomRes) Init(room *GameRoom, userId UserId, balance Amount) *ClientJoinRoomRes {
	res.CMD = CMD_JOIN_ROOM_SUCCESS
	res.RoomId = room.RoomId
	res.GameId = room.GameId
	res.Balance = balance
	res.RoomInfo = (&RoomInfoContent{}).Init(room, userId)
	return res
}

type RoomInfoContent struct {
	GameNumber      GameNumber
	RoomId          RoomId
	RoundId         RoundId
	SeatId          SeatId
	GameState       GameState
	RemainStateTime float64
	TotalBetTime    float64
	PlayerBets      [][]*BetPlace
	StateStartTime  int64
	ServerTime      int64

	GameInitData interface{}

	PayoutContent *PayoutContent
	Result        interface{}

	Txh string
	W   string
}

func (Content *RoomInfoContent) Init(room *GameRoom, userId UserId) *RoomInfoContent {
	game := GetGameInterface(room.GameId, room.Server)
	Content.GameNumber = game.GetGameNumber()
	Content.RoomId = room.RoomId
	Content.RoundId = game.GetRoundId()
	Content.GameState = game.GetCurState()
	Content.SeatId = 0
	Content.RemainStateTime = game.GetRemainStateTime()
	Content.TotalBetTime = game.GetTotalBetTime()
	Content.StateStartTime = game.GetStateStartTime()
	Content.ServerTime = time.Now().UnixMilli()
	Content.Result = game.GetResultData()
	Content.Txh = game.GetTxh()
	Content.W = game.GetW()

	betState := []*BetPlace{}

	Content.PayoutContent = nil
	betInfo, has := room.BetInfosMap[userId]
	if has {
		betState = betInfo.ConfirmedBetState
		Content.PayoutContent = &PayoutContent{}

		payouts := []*PayoutInfo{}
		// slice array
		payouts = append(payouts, betInfo.ConfirmedPayouts...)
		playerPayout := PlayerPayout{Payouts: &payouts}

		Content.PayoutContent.PlayerPayouts = &[]*PlayerPayout{&playerPayout}

		Content.PayoutContent.Balance = TruncateAmount(betInfo.Balance)
	}

	Content.PlayerBets = [][]*BetPlace{betState}

	return Content
}

type ClientRoomInfoResponse struct {
	CMD     string
	RoomId  RoomId
	Content *RoomInfoContent
}

func (res *ClientRoomInfoResponse) Init(room *GameRoom, userId UserId, gameInitData interface{}) *ClientRoomInfoResponse {
	res.CMD = CMD_ROOM_INFO
	res.RoomId = room.RoomId
	res.Content = (&RoomInfoContent{}).Init(room, userId)
	res.Content.GameInitData = gameInitData

	return res
}

type ClientTrendResponse struct {
	CMD    string
	GameId GameId
	RoomId RoomId
	Trends *[]*TrendItem
}

func (res *ClientTrendResponse) Init(room *GameRoom, Trends *[]*TrendItem) *ClientTrendResponse {
	res.CMD = CMD_TRENDS

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	res.Trends = Trends
	return res
}

type PlayerPayout struct {
	Payouts *[]*PayoutInfo
}

type PayoutContent struct {
	Balance       Amount
	PlayerPayouts *[]*PlayerPayout
}

type ClientPayoutResponse struct {
	CMD           string
	GameNumber    GameNumber
	GameId        GameId
	RoomId        RoomId
	RoundId       RoundId
	PayoutContent *PayoutContent
}

func (res *ClientPayoutResponse) Init(room *GameRoom, PlayerPayouts *[]*PlayerPayout, Balance Amount) *ClientPayoutResponse {
	res.CMD = CMD_PAYOUT_SUCCESS
	game := GetGameInterface(room.GameId, room.Server)
	res.GameNumber = game.GetGameNumber()
	res.GameId = room.GameId
	res.RoomId = room.RoomId
	res.RoundId = game.GetRoundId()
	res.PayoutContent = &PayoutContent{}
	res.PayoutContent.Balance = TruncateAmount(Balance)
	res.PayoutContent.PlayerPayouts = PlayerPayouts
	return res
}
