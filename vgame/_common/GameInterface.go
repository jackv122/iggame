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
	GetStateStartTime() int64
	GetTotalBetTime() float64

	SaveGameState()
	LoadGameState() bool
	// for saving to data base
	GetGameResultString() string
	GetResultData() interface{}
	GetGenResultData() interface{}
	GetTxh() string
	GetW() string
}
