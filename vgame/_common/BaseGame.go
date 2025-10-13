package com

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// BaseGame contains common properties and methods shared by all games
type BaseGame struct {
	Server     *GameServer
	GameId     GameId
	Name       string
	TimeKeeper *TimeKeeper
	RoomList   []*GameRoom

	RoundId    RoundId
	TickTime   float64
	GameNumber GameNumber

	Txh      string
	W        string
	StateMng *StateManager

	Trends []*TrendItem

	TickCount int

	// Game data maps
	// can map a BetType or BetKind to a payout ratio
	PayoutMap         map[string]Amount
	SmallLimitBetMap  map[Currency]map[BetKind]*BetLimit
	MediumLimitBetMap map[Currency]map[BetKind]*BetLimit
	BetKindMap        map[string]BetKind
}

// Common initialization method
func (g *BaseGame) InitBase(server *GameServer, gameId GameId, name string) {
	g.Server = server
	g.GameId = gameId
	g.Name = name
	g.RoundId = 0
	g.TimeKeeper = &TimeKeeper{}
	g.RoomList = []*GameRoom{}
	g.TickTime = 0
	g.TickCount = 0
	g.Trends = []*TrendItem{}
	g.Txh = ""
	g.W = ""

	// Initialize game data maps
	g.PayoutMap = make(map[string]Amount)
	g.SmallLimitBetMap = make(map[Currency]map[BetKind]*BetLimit)
	g.MediumLimitBetMap = make(map[Currency]map[BetKind]*BetLimit)
	g.BetKindMap = make(map[string]BetKind)

	// Load limit maps from JSON configuration
	g.loadLimitMapsFromJSON()
}

// Common methods that all games share
func (g *BaseGame) GetTimeKeeper() *TimeKeeper {
	return g.TimeKeeper
}

func (g *BaseGame) GetGameId() GameId {
	return g.GameId
}

func (g *BaseGame) GetName() string {
	return g.Name
}

func (g *BaseGame) GetRoundId() RoundId {
	return g.RoundId
}

func (g *BaseGame) GetGameNumber() GameNumber {
	return g.GameNumber
}

func (g *BaseGame) GetTrends() []*TrendItem {
	return g.Trends
}

func (g *BaseGame) GetTrendsByPage(page uint32) []*TrendItem {
	start := page * uint32(TREND_PAGE_SIZE)
	end := start + uint32(TREND_PAGE_SIZE)
	if start >= uint32(len(g.Trends)) {
		return []*TrendItem{}
	}
	if end > uint32(len(g.Trends)) {
		end = uint32(len(g.Trends))
	}
	return g.Trends[start:end]
}

func (g *BaseGame) GetW() string {
	return g.W
}

func (g *BaseGame) GetRemainStateTime() float64 {
	return g.StateMng.StateDurs[g.StateMng.CurrState] - g.StateMng.StateTime
}

func (g *BaseGame) GetStateStartTime() int64 {
	return g.StateMng.StateStartTime
}

func (g *BaseGame) GetTotalBetTime() float64 {
	return g.StateMng.StateDurs[GAME_STATE_BETTING]
}

func (g *BaseGame) InitRoomForGame(room *GameRoom) {
	g.RoomList = append(g.RoomList, room)
	// init limit map
}

func (g *BaseGame) LoadTrends(gameId GameId, page uint32) []*TrendItem {
	return g.Trends
}

func (g *BaseGame) GetCurState() GameState {
	return g.StateMng.CurrState
}

func (g *BaseGame) GetTxh() string {
	return g.Txh
}

// Common utility methods

func (g *BaseGame) Update(dt float64) {
	g.StateMng.StateUpdate(dt)
	g.TickTime += dt
	if g.TickTime > 1.0 {
		g.OnTick()
		g.TickTime -= 1.0
	}
}

func (g *BaseGame) OnTick() {
	switch g.StateMng.CurrState {
	case GAME_STATE_BETTING:
		remainBettingTime := uint16(g.StateMng.StateDurs[1] - g.StateMng.StateTime)
		for _, room := range g.RoomList {
			res := (&TickResponse{}).Init(room)
			res.Time = remainBettingTime
			res.RoomTotalBet = float64(room.GetRoomTotalBet())
			room.BroadcastMessage(res)
		}
	}
	g.TickCount++
	if g.TickCount > 0 {
		g.TickCount = 0
		for _, room := range g.RoomList {
			room.NotifyRoomStats()
		}
	}
}

func (g *BaseGame) Stop() {
	fmt.Printf("%s stop\n", g.Name)
	g.StateMng.ResetState()
}

// Common state entry methods that are the same for all games
func (g *BaseGame) OnEnterBetting() {
	fmt.Printf("%s entering BETTING state\n", g.Name)
	for _, room := range g.RoomList {
		res := (&BaseGameResponse{}).Init(room, CMD_START_BET_SUCCEED)
		room.BroadcastMessage(res)
	}
}

func (g *BaseGame) OnEnterCloseBetting() {
	fmt.Printf("%s entering CLOSE_BETTING state\n", g.Name)
	for _, room := range g.RoomList {
		res := (&BaseGameResponse{}).Init(room, CMD_STOP_BET_SUCCEED)
		room.BroadcastMessage(res)
	}
}

func (g *BaseGame) OnEnterPayout() {
	fmt.Printf("%s entering PAYOUT state\n", g.Name)
	// broadcast payout to users ---
	for _, room := range g.RoomList {
		room.NotifyReward()
	}
}

func (g *BaseGame) GetBetKind(betType BetType) BetKind {
	return g.BetKindMap[string(betType)]
}

func (game *BaseGame) GetAllBetLimits() []map[Currency]map[BetKind]*BetLimit {
	limits := []map[Currency]map[BetKind]*BetLimit{game.SmallLimitBetMap, game.MediumLimitBetMap}
	return limits
}

func (g *BaseGame) GetBetLimit(level LimitLevel) map[Currency]map[BetKind]*BetLimit {
	switch level {
	case LIMIT_LEVEL_SMALL:
		return g.SmallLimitBetMap
	case LIMIT_LEVEL_MEDIUM:
		return g.MediumLimitBetMap
	}

	return g.SmallLimitBetMap
}

func (g *BaseGame) GetPayout(betKind BetKind) Amount {
	return g.PayoutMap[string(betKind)]
}

// loadLimitMapsFromJSON loads limit maps from JSON configuration file
func (g *BaseGame) loadLimitMapsFromJSON() {
	configPath := fmt.Sprintf("config/%s_limits.json", g.GameId)

	// Check if config file exists, if not use default values
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file %s not found, using default values\n", configPath)
		g.loadDefaultLimitMaps()
		return
	}

	// Read and parse JSON config
	jsonData, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file %s: %v, using default values\n", configPath, err)
		g.loadDefaultLimitMaps()
		return
	}

	var configData map[string]interface{}
	err = json.Unmarshal(jsonData, &configData)
	if err != nil {
		fmt.Printf("Error parsing config file %s: %v, using default values\n", configPath, err)
		g.loadDefaultLimitMaps()
		return
	}

	// Load small limit bet map
	if smallLimitData, ok := configData["smallLimitBetMap"].(map[string]interface{}); ok {
		for currency, currencyData := range smallLimitData {
			if currencyMap, ok := currencyData.(map[string]interface{}); ok {
				betMap := map[BetKind]*BetLimit{}
				for betKindStr, limitData := range currencyMap {
					if limitMap, ok := limitData.(map[string]interface{}); ok {
						if minVal, ok := limitMap["min"].(float64); ok {
							if maxVal, ok := limitMap["max"].(float64); ok {
								betMap[BetKind(betKindStr)] = &BetLimit{
									Min: Amount(minVal),
									Max: Amount(maxVal),
								}
							}
						}
					}
				}
				g.SmallLimitBetMap[Currency(currency)] = betMap
			}
		}
	}

	// Load medium limit bet map
	if mediumLimitData, ok := configData["mediumLimitBetMap"].(map[string]interface{}); ok {
		for currency, currencyData := range mediumLimitData {
			if currencyMap, ok := currencyData.(map[string]interface{}); ok {
				betMap := map[BetKind]*BetLimit{}
				for betKindStr, limitData := range currencyMap {
					if limitMap, ok := limitData.(map[string]interface{}); ok {
						if minVal, ok := limitMap["min"].(float64); ok {
							if maxVal, ok := limitMap["max"].(float64); ok {
								betMap[BetKind(betKindStr)] = &BetLimit{
									Min: Amount(minVal),
									Max: Amount(maxVal),
								}
							}
						}
					}
				}
				g.MediumLimitBetMap[Currency(currency)] = betMap
			}
		}
	}

	fmt.Printf("Configuration loaded from JSON file: %s\n", configPath)
}

// loadDefaultLimitMaps loads default limit maps when JSON config is not available
func (g *BaseGame) loadDefaultLimitMaps() {
	panic("load Limit Maps failed")
}
