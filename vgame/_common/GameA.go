package com

type GameA struct {
	BaseGame
	payedout  bool
	isStarted bool
}

func (g *GameA) Start() {

}

func (g *GameA) InitRoomForGame(room *GameRoom) {

}

// work as constructor
func (g *GameA) Init(server *GameServer) *GameA {
	g.InitBase(server, IDGameA, "GameA")
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

func (game *GameA) OnMessage(roomId RoomId, connId ConnectionId, msg string) {
	//fmt.Printf("game: %s OnMessage: %s\n", game.name, *msg)
}

func (game *GameA) Stop() {
	if game.TimeKeeper.Timer != nil {
		game.TimeKeeper.Timer.Stop()
	}
}

func (game *GameA) GetRoundId() RoundId {
	return 1
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
	return game.Trends
}

func (game *GameA) GetResultString() string {
	return ""
}
