package cock

import (
	com "vgame/_common"
)

const (
	BET_Straight = 0
	GAME_VERSION = "1.0.0"
)

type CockID com.BetType

const (
	COCK_001 CockID = "001"
	COCK_002 CockID = "002"
	COCK_003 CockID = "003"
	COCK_004 CockID = "004"
)

type GameInitData struct {
	Version string
	Cock_1  *CockData
	Cock_2  *CockData
}

type GameResultData struct {
	Version string
	Winner  CockID
}

type CockStrategyData struct {
}

func (d *CockStrategyData) init(g *CockStrategy) *CockStrategyData {

	// payout
	g.PayoutMap[com.BetKind(BET_Straight)] = 1 + 0.95

	// init BetKindMap
	g.BetKindMap[string(COCK_001)] = com.BetKind(BET_Straight)
	g.BetKindMap[string(COCK_002)] = com.BetKind(BET_Straight)
	g.BetKindMap[string(COCK_003)] = com.BetKind(BET_Straight)
	g.BetKindMap[string(COCK_004)] = com.BetKind(BET_Straight)

	return d
}
