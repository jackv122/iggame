package com

type GameInterface interface {
	Start()
	Update(dt float64)
	OnMessage(RoomId, ConnectionId, string)
	Stop()
	GetTimeKeeper() *TimeKeeper
	InitRoomForGame(room *GameRoom)
	GetRoundId() RoundId
	GetGameNumber() GameNumber
	GetBetKind(betType BetType) BetKind
	GetBetLimit(level LimitLevel) map[Currency]map[BetKind]*BetLimit
	GetAllBetLimits() [](map[Currency]map[BetKind]*BetLimit)
	LoadTrends(GameId GameId, page uint32) []*TrendItem

	GetCurState() GameState
	GetRemainStateTime() float64
	GetTotalBetTime() float64

	SaveGameState()
	LoadGameState() bool
	GetGameResult() string
	GetResultString() string
	GetTxh() string
	GetW() string
}
