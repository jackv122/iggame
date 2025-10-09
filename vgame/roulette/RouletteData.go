package roul

import (
	"strconv"
	com "vgame/_common"
)

const (
	BET_Straight  = "0"
	BET_Split     = "1"
	BET_Street    = "2"
	BET_Corner    = "3"
	BET_Line      = "4"
	BET_Trio      = "5"
	BET_Basket    = "6"
	BET_Odd_Even  = "7"
	BET_Red_Black = "8"
	BET_High_Low  = "9"
	BET_Columns   = "10"
	BET_Dozens    = "11"
)

type RouletteData struct {
	betResultMap map[string]map[int]bool
}

func (d *RouletteData) init(g *Roulette) *RouletteData {
	d.initPayoutMap(g)
	d.initGameDatas(g)
	return d
}

func (d *RouletteData) initPayoutMap(g *Roulette) *map[string]com.Amount {
	payoutMap := g.PayoutMap
	payoutMap[BET_Straight] = 35 + 1
	payoutMap[BET_Split] = 17 + 1
	payoutMap[BET_Street] = 11 + 1
	payoutMap[BET_Corner] = 8 + 1
	payoutMap[BET_Line] = 5 + 1
	payoutMap[BET_Trio] = 11 + 1
	payoutMap[BET_Basket] = 6 + 1
	payoutMap[BET_Odd_Even] = 1 + 1
	payoutMap[BET_Red_Black] = 1 + 1
	payoutMap[BET_High_Low] = 1 + 1
	payoutMap[BET_Columns] = 2 + 1
	payoutMap[BET_Dozens] = 2 + 1
	return &payoutMap
}

// init betResultMap and betKindMap, return betKindMap
func (d *RouletteData) initGameDatas(g *Roulette) {
	d.betResultMap = map[string]map[int]bool{}
	hlMap := d.betResultMap
	betKindMap := map[string]com.BetKind{}

	var m map[int]bool = nil
	// Straight F
	for i := 0; i <= 36; i++ {
		m = map[int]bool{}
		m[i] = true
		hlMap["F"+strconv.Itoa(i)] = m
		betKindMap["F"+strconv.Itoa(i)] = BET_Straight
	}

	// Split G
	for i := 0; i < 36; i++ {
		m = map[int]bool{}
		if i < 3 {
			m[0] = true
			m[i+1] = true
		} else {
			m = map[int]bool{}
			m[i-2] = true
			m[i+1] = true
		}
		hlMap["G"+strconv.Itoa(i)] = m
		betKindMap["G"+strconv.Itoa(i)] = BET_Split
	}

	// Split H
	for i := 0; i < 12; i++ {
		k := i * 3
		// top Splits
		m = map[int]bool{}
		m[k+2] = true
		m[k+3] = true
		hlMap["H"+strconv.Itoa(i*2+1)] = m
		betKindMap["H"+strconv.Itoa(i*2+1)] = BET_Split

		// bottom Splits
		m = map[int]bool{}
		m[k+1] = true
		m[k+2] = true
		hlMap["H"+strconv.Itoa(i*2)] = m
		betKindMap["H"+strconv.Itoa(i*2)] = BET_Split
	}

	// Corner I
	for i := 0; i < 11; i++ {
		k := i * 3
		// top
		m = map[int]bool{}
		m[k+2] = true
		m[k+3] = true
		m[k+5] = true
		m[k+6] = true
		hlMap["I"+strconv.Itoa(i*2+1)] = m
		betKindMap["I"+strconv.Itoa(i*2+1)] = BET_Corner

		// bottom
		m = map[int]bool{}
		m[k+1] = true
		m[k+2] = true
		m[k+4] = true
		m[k+5] = true
		hlMap["I"+strconv.Itoa(i*2)] = m
		betKindMap["I"+strconv.Itoa(i*2)] = BET_Corner
	}

	// init bet type Street J -----
	for i := 0; i < 12; i++ {
		k := i * 3
		// top
		m = map[int]bool{}
		m[k+1] = true
		m[k+2] = true
		m[k+3] = true
		hlMap["J"+strconv.Itoa(i)] = m
		betKindMap["J"+strconv.Itoa(i)] = BET_Street
	}

	// init bet type Line K -----
	for i := 0; i < 11; i++ {
		m = map[int]bool{}
		for j := i*3 + 1; j < i*3+1+6; j++ {
			m[j] = true
		}
		hlMap["K"+strconv.Itoa(i)] = m
		betKindMap["K"+strconv.Itoa(i)] = BET_Line
	}

	// init bet type Trio L -----
	m = map[int]bool{}
	m[0] = true
	m[1] = true
	m[2] = true
	hlMap["L0"] = m
	betKindMap["L0"] = BET_Trio

	m = map[int]bool{}
	m[0] = true
	m[2] = true
	m[3] = true
	hlMap["L1"] = m
	betKindMap["L1"] = BET_Trio

	// init bet type Basket M -----
	m = map[int]bool{}
	m[0] = true
	m[1] = true
	m[2] = true
	m[3] = true
	hlMap["M"] = m
	betKindMap["M"] = BET_Basket

	// init bet type Dozens A -----
	for i := 0; i < 3; i++ {
		m = map[int]bool{}
		for j := 1; j <= 12; j++ {
			m[i*12+j] = true
		}
		hlMap["A"+strconv.Itoa(i)] = m
		betKindMap["A"+strconv.Itoa(i)] = BET_Dozens
	}

	// init bet type Red / Black B -----
	hlMap["B0"] = map[int]bool{}
	betKindMap["B0"] = BET_Red_Black
	hlMap["B1"] = map[int]bool{}
	betKindMap["B1"] = BET_Red_Black
	red := []int{1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36}
	for _, i := range red {
		hlMap["B0"][i] = true
	}
	blk := []int{2, 4, 6, 8, 10, 11, 13, 15, 17, 20, 22, 24, 26, 28, 29, 31, 33, 35}
	for _, i := range blk {
		hlMap["B1"][i] = true
	}

	// init bet type Even / Odd C -----
	hlMap["C0"] = map[int]bool{}
	betKindMap["C0"] = BET_Odd_Even
	hlMap["C1"] = map[int]bool{}
	betKindMap["C1"] = BET_Odd_Even
	for i := 1; i <= 36; i++ {
		if i%2 == 0 {
			hlMap["C0"][i] = true
		} else {
			hlMap["C1"][i] = true
		}
	}

	// init bet type High /  Low D -----
	hlMap["D0"] = map[int]bool{}
	betKindMap["D0"] = BET_High_Low
	hlMap["D1"] = map[int]bool{}
	betKindMap["D1"] = BET_High_Low
	for i := 1; i <= 36; i++ {
		if i <= 18 {
			hlMap["D0"][i] = true
		} else {
			hlMap["D1"][i] = true
		}
	}

	// init bet type Columns E -----
	for i := 0; i < 3; i++ {
		m = map[int]bool{}
		for j := 0; j < 12; j++ {
			m[j*3+i+1] = true
		}
		hlMap["E"+strconv.Itoa(i)] = m
		betKindMap["E"+strconv.Itoa(i)] = BET_Columns
	}

	g.BetKindMap = betKindMap
}
