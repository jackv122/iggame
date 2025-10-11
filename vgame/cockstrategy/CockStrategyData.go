package cock

import (
	com "vgame/_common"
)

const (
	// bet kind
	BET_Straight = "0"
	GAME_VERSION = "1.0.0"
)

type CockID com.BetType

const (
	COCK_001 CockID = "c01"
	COCK_002 CockID = "c02"
	COCK_003 CockID = "c03"
	COCK_004 CockID = "c04"
)

const (
	BET_TYPE_LEFT  com.BetType = "0"
	BET_TYPE_RIGHT com.BetType = "1"
)

type GameInitData struct {
	Version string
	Cock_1  *CockData
	Cock_2  *CockData
}

type GameResultData struct {
	Version        string
	Winner         CockID
	HighlightGates []com.BetType
}

type CockStrategyData struct {
	betResultMap map[com.BetType]bool
}

func (d *CockStrategyData) init(g *CockStrategy) *CockStrategyData {
	d.betResultMap = map[com.BetType]bool{}
	// init BetKindMap
	g.BetKindMap[string(BET_TYPE_LEFT)] = com.BetKind(BET_Straight)
	g.BetKindMap[string(BET_TYPE_RIGHT)] = com.BetKind(BET_Straight)

	return d
}
