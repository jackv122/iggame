package roul

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	com "vgame/_common"
)

const (
	BET_Straight  = 0
	BET_Split     = 1
	BET_Street    = 2
	BET_Corner    = 3
	BET_Line      = 4
	BET_Trio      = 5
	BET_Basket    = 6
	BET_Odd_Even  = 7
	BET_Red_Black = 8
	BET_High_Low  = 9
	BET_Columns   = 10
	BET_Dozens    = 11
)

type RouletteData struct {
	betResultMap map[string]map[int]bool
	// map a betId to a bet kind
	BetKindMap        map[string]com.BetKind
	PayoutMap         map[com.BetKind]com.Amount
	SmallLimitBetMap  map[com.Currency]map[com.BetKind]*com.BetLimit
	MediumLimitBetMap map[com.Currency]map[com.BetKind]*com.BetLimit
}

func (d *RouletteData) init() *RouletteData {
	d.initBetKindMap()
	d.PayoutMap = map[com.BetKind]com.Amount{}
	d.PayoutMap[BET_Straight] = 35 + 1
	d.PayoutMap[BET_Split] = 17 + 1
	d.PayoutMap[BET_Street] = 11 + 1
	d.PayoutMap[BET_Corner] = 8 + 1
	d.PayoutMap[BET_Line] = 5 + 1
	d.PayoutMap[BET_Trio] = 11 + 1
	d.PayoutMap[BET_Basket] = 6 + 1
	d.PayoutMap[BET_Odd_Even] = 1 + 1
	d.PayoutMap[BET_Red_Black] = 1 + 1
	d.PayoutMap[BET_High_Low] = 1 + 1
	d.PayoutMap[BET_Columns] = 2 + 1
	d.PayoutMap[BET_Dozens] = 2 + 1

	// Load limit maps from JSON configuration
	d.loadLimitMapsFromJSON()
	return d
}

func (d *RouletteData) initBetKindMap() {
	d.betResultMap = map[string]map[int]bool{}
	d.BetKindMap = map[string]com.BetKind{}
	hlMap := d.betResultMap
	var m map[int]bool = nil
	// Straight F
	for i := 0; i <= 36; i++ {
		m = map[int]bool{}
		m[i] = true
		hlMap["F"+strconv.Itoa(i)] = m
		d.BetKindMap["F"+strconv.Itoa(i)] = BET_Straight
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
		d.BetKindMap["G"+strconv.Itoa(i)] = BET_Split
	}

	// Split H
	for i := 0; i < 12; i++ {
		k := i * 3
		// top Splits
		m = map[int]bool{}
		m[k+2] = true
		m[k+3] = true
		hlMap["H"+strconv.Itoa(i*2+1)] = m
		d.BetKindMap["H"+strconv.Itoa(i*2+1)] = BET_Split

		// bottom Splits
		m = map[int]bool{}
		m[k+1] = true
		m[k+2] = true
		hlMap["H"+strconv.Itoa(i*2)] = m
		d.BetKindMap["H"+strconv.Itoa(i*2)] = BET_Split
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
		d.BetKindMap["I"+strconv.Itoa(i*2+1)] = BET_Corner

		// bottom
		m = map[int]bool{}
		m[k+1] = true
		m[k+2] = true
		m[k+4] = true
		m[k+5] = true
		hlMap["I"+strconv.Itoa(i*2)] = m
		d.BetKindMap["I"+strconv.Itoa(i*2)] = BET_Corner
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
		d.BetKindMap["J"+strconv.Itoa(i)] = BET_Street
	}

	// init bet type Line K -----
	for i := 0; i < 11; i++ {
		m = map[int]bool{}
		for j := i*3 + 1; j < i*3+1+6; j++ {
			m[j] = true
		}
		hlMap["K"+strconv.Itoa(i)] = m
		d.BetKindMap["K"+strconv.Itoa(i)] = BET_Line
	}

	// init bet type Trio L -----
	m = map[int]bool{}
	m[0] = true
	m[1] = true
	m[2] = true
	hlMap["L0"] = m
	d.BetKindMap["L0"] = BET_Trio

	m = map[int]bool{}
	m[0] = true
	m[2] = true
	m[3] = true
	hlMap["L1"] = m
	d.BetKindMap["L1"] = BET_Trio

	// init bet type Basket M -----
	m = map[int]bool{}
	m[0] = true
	m[1] = true
	m[2] = true
	m[3] = true
	hlMap["M"] = m
	d.BetKindMap["M"] = BET_Basket

	// init bet type Dozens A -----
	for i := 0; i < 3; i++ {
		m = map[int]bool{}
		for j := 1; j <= 12; j++ {
			m[i*12+j] = true
		}
		hlMap["A"+strconv.Itoa(i)] = m
		d.BetKindMap["A"+strconv.Itoa(i)] = BET_Dozens
	}

	// init bet type Red / Black B -----
	hlMap["B0"] = map[int]bool{}
	d.BetKindMap["B0"] = BET_Red_Black
	hlMap["B1"] = map[int]bool{}
	d.BetKindMap["B1"] = BET_Red_Black
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
	d.BetKindMap["C0"] = BET_Odd_Even
	hlMap["C1"] = map[int]bool{}
	d.BetKindMap["C1"] = BET_Odd_Even
	for i := 1; i <= 36; i++ {
		if i%2 == 0 {
			hlMap["C0"][i] = true
		} else {
			hlMap["C1"][i] = true
		}
	}

	// init bet type High /  Low D -----
	hlMap["D0"] = map[int]bool{}
	d.BetKindMap["D0"] = BET_High_Low
	hlMap["D1"] = map[int]bool{}
	d.BetKindMap["D1"] = BET_High_Low
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
		d.BetKindMap["E"+strconv.Itoa(i)] = BET_Columns
	}
}

func (d *RouletteData) loadLimitMapsFromJSON() {
	configPath := "config/roulette_limits.json"

	// Check if config file exists, if not use default values
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Roulette config file %s not found, using default values\n", configPath)
		d.loadDefaultLimitMaps()
		return
	}

	// Read and parse JSON config
	jsonData, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading roulette config file: %v, using default values\n", err)
		d.loadDefaultLimitMaps()
		return
	}

	var configData map[string]interface{}
	err = json.Unmarshal(jsonData, &configData)
	if err != nil {
		fmt.Printf("Error parsing roulette config file: %v, using default values\n", err)
		d.loadDefaultLimitMaps()
		return
	}

	// Initialize maps
	d.SmallLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	d.MediumLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}

	// Load small limit bet map
	if smallLimitData, ok := configData["smallLimitBetMap"].(map[string]interface{}); ok {
		for currency, currencyData := range smallLimitData {
			if currencyMap, ok := currencyData.(map[string]interface{}); ok {
				betMap := map[com.BetKind]*com.BetLimit{}
				for betKindStr, limitData := range currencyMap {
					if betKindInt, err := strconv.Atoi(betKindStr); err == nil {
						if limitMap, ok := limitData.(map[string]interface{}); ok {
							if minVal, ok := limitMap["min"].(float64); ok {
								if maxVal, ok := limitMap["max"].(float64); ok {
									betMap[com.BetKind(betKindInt)] = &com.BetLimit{
										Min: com.Amount(minVal),
										Max: com.Amount(maxVal),
									}
								}
							}
						}
					}
				}
				d.SmallLimitBetMap[com.Currency(currency)] = betMap
			}
		}
	}

	// Load medium limit bet map
	if mediumLimitData, ok := configData["mediumLimitBetMap"].(map[string]interface{}); ok {
		for currency, currencyData := range mediumLimitData {
			if currencyMap, ok := currencyData.(map[string]interface{}); ok {
				betMap := map[com.BetKind]*com.BetLimit{}
				for betKindStr, limitData := range currencyMap {
					if betKindInt, err := strconv.Atoi(betKindStr); err == nil {
						if limitMap, ok := limitData.(map[string]interface{}); ok {
							if minVal, ok := limitMap["min"].(float64); ok {
								if maxVal, ok := limitMap["max"].(float64); ok {
									betMap[com.BetKind(betKindInt)] = &com.BetLimit{
										Min: com.Amount(minVal),
										Max: com.Amount(maxVal),
									}
								}
							}
						}
					}
				}
				d.MediumLimitBetMap[com.Currency(currency)] = betMap
			}
		}
	}

	fmt.Println("Roulette configuration loaded from JSON file")
}

func (d *RouletteData) loadDefaultLimitMaps() {
	d.SmallLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	usdc := map[com.BetKind]*com.BetLimit{}

	usdc[BET_Straight] = &com.BetLimit{Min: 0.1, Max: 3}
	usdc[BET_Split] = &com.BetLimit{Min: 0.1, Max: 6}
	usdc[BET_Street] = &com.BetLimit{Min: 0.1, Max: 9}
	usdc[BET_Corner] = &com.BetLimit{Min: 0.1, Max: 12}
	usdc[BET_Line] = &com.BetLimit{Min: 0.1, Max: 18}
	usdc[BET_Trio] = &com.BetLimit{Min: 0.1, Max: 9}
	usdc[BET_Basket] = &com.BetLimit{Min: 0.1, Max: 16}
	usdc[BET_Odd_Even] = &com.BetLimit{Min: 0.1, Max: 54}
	usdc[BET_Red_Black] = &com.BetLimit{Min: 0.1, Max: 54}
	usdc[BET_High_Low] = &com.BetLimit{Min: 0.1, Max: 54}
	usdc[BET_Columns] = &com.BetLimit{Min: 0.1, Max: 36}
	usdc[BET_Dozens] = &com.BetLimit{Min: 0.1, Max: 36}
	d.SmallLimitBetMap[com.Currency("USDC")] = usdc

	d.MediumLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	usdc = map[com.BetKind]*com.BetLimit{}
	usdc[BET_Straight] = &com.BetLimit{Min: 1, Max: 30}
	usdc[BET_Split] = &com.BetLimit{Min: 1, Max: 60}
	usdc[BET_Street] = &com.BetLimit{Min: 1, Max: 90}
	usdc[BET_Corner] = &com.BetLimit{Min: 1, Max: 120}
	usdc[BET_Line] = &com.BetLimit{Min: 1, Max: 180}
	usdc[BET_Trio] = &com.BetLimit{Min: 1, Max: 90}
	usdc[BET_Basket] = &com.BetLimit{Min: 1, Max: 160}
	usdc[BET_Odd_Even] = &com.BetLimit{Min: 1, Max: 540}
	usdc[BET_Red_Black] = &com.BetLimit{Min: 1, Max: 540}
	usdc[BET_High_Low] = &com.BetLimit{Min: 1, Max: 540}
	usdc[BET_Columns] = &com.BetLimit{Min: 1, Max: 360}
	usdc[BET_Dozens] = &com.BetLimit{Min: 1, Max: 360}
	d.MediumLimitBetMap[com.Currency("USDC")] = usdc
}
