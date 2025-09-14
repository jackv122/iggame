package com

type GameA struct {
	server     *GameServer
	gameId     GameId
	name       string
	timeKeeper *TimeKeeper

	payedout   bool
	gameNumber GameNumber // need to gen on state STARTING
	isStarted  bool
	trends     []*TrendItem
}

func (g *GameA) Start() {

}

func (g *GameA) InitRoomForGame(room *GameRoom) {

}

// work as constructor
func (g *GameA) Init(server *GameServer) *GameA {
	g.server = server
	g.gameId = IDGameA
	g.name = "GameA"
	g.timeKeeper = &TimeKeeper{}
	g.trends = []*TrendItem{}
	return g
}

func (g *GameA) SaveGameState() {}
func (g *GameA) LoadGameState() bool {
	return false
}

func (game *GameA) Update(dt float64) {
	//time.Sleep(1 * time.Second)

	//fmt.Println("game Update: ", game.name)
}

func (game *GameA) OnMessage(connId ConnectionId, msg string) {
	//fmt.Printf("game: %s OnMessage: %s\n", game.name, *msg)
}

func (game *GameA) Stop() {
	if game.timeKeeper.Timer != nil {
		game.timeKeeper.Timer.Stop()
	}
}

func (game *GameA) GetTimeKeeper() *TimeKeeper {
	return game.timeKeeper
}

func (game *GameA) GetRoundId() RoundId {
	return 1
}

func (game *GameA) GetGameNumber() GameNumber {
	return game.gameNumber
}

func (game *GameA) GetBetKind(betType BetType) BetKind {
	return BET_COMMON
}

func (game *GameA) GetBetLimit(level LimitLevel) map[Currency]map[BetKind]*BetLimit {
	return nil
}

func (game *GameA) GetAllBetLimits() []map[Currency]map[BetKind]*BetLimit {
	return []map[Currency]map[BetKind]*BetLimit{}
}

func (game *GameA) GetCurState() GameState {
	return GAME_STATE_STARTING
}

func (game *GameA) GetRemainStateTime() float64 {
	return 0
}

func (game *GameA) GetTotalBetTime() float64 {
	return 1
}

func (game *GameA) GetGameResult() string {
	return ""
}

func (game *GameA) LoadTrends(gameId GameId, page uint32) []*TrendItem {
	return game.trends
}

func (game *GameA) GetTxh() string {
	return ""
}

func (game *GameA) GetW() string {
	return ""
}

func (game *GameA) GetResultString() string {
	return ""
}
