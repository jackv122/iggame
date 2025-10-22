package cock

import (
	com "vgame/_common"
)

const (
	// bet kind
	BET_Straight = "0"
)

var GAME_VERSION = "1.0.0"

type CockID com.BetType

const (
	COCK_01 CockID = "c01"
	COCK_02 CockID = "c02"
	COCK_03 CockID = "c03"
	COCK_04 CockID = "c04"
)

const (
	BET_TYPE_LEFT  com.BetType = "0"
	BET_TYPE_RIGHT com.BetType = "1"
)

type GameStateData struct {
	Version           string
	Cock_1            *CockData
	Cock_2            *CockData
	PairIndex         int
	ResultBattleIndex int
	GenResultData     *GenResultContent
}

type GameInitDataRes struct {
	Version string
	Cock_1  *CockData
	Cock_2  *CockData
}

func (data *GameStateData) GetInitData() GameInitDataRes {
	return GameInitDataRes{
		Version: data.Version,
		Cock_1:  data.Cock_1,
		Cock_2:  data.Cock_2,
	}
}

type GameResultData struct {
	Version        string
	Winner         CockID
	HighlightGates []com.BetType
}

type CockConfig struct {
	Name   string     `json:"name"`
	ID     CockID     `json:"id"`
	S      float64    `json:"s"`
	A      float64    `json:"a"`
	Payout com.Amount `json:"payout"`
}

type Stats struct {
	Version         string         `json:"version"`
	Total           int            `json:"total"`
	Win             map[string]int `json:"win"`
	FullWin         map[string]int `json:"fullWin"`
	MinDur          float64        `json:"minDur"`
	MaxDur          float64        `json:"maxDur"`
	LeftCockConfig  CockConfig     `json:"leftCockConfig"`
	RightCockConfig CockConfig     `json:"rightCockConfig"`
}

type BattleConfig struct {
	Stats Stats
	DB    []string
}

type CockStrategyData struct {
	betResultMap  map[com.BetType]bool
	BattleConfigs []BattleConfig
}

type BattleInfo struct {
	Randoms   []string `json:"randoms"`
	Winner    string   `json:"winner"`
	Index     int      `json:"index"`
	Duration  float64  `json:"duration"`
	IsFullWin bool     `json:"isFullWin"`
}

func (d *CockStrategyData) init(g *CockStrategy) *CockStrategyData {
	d.betResultMap = map[com.BetType]bool{}
	// init BetKindMap
	g.BetKindMap[string(BET_TYPE_LEFT)] = com.BetKind(BET_Straight)
	g.BetKindMap[string(BET_TYPE_RIGHT)] = com.BetKind(BET_Straight)

	return d
}
