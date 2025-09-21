package cock

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	com "vgame/_common"
)

const (
	BET_Straight = 0
)

type CockID com.BetType

const (
	COCK_001 CockID = "001"
	COCK_002 CockID = "002"
	COCK_003 CockID = "003"
	COCK_004 CockID = "004"
)

type CockStrategyData struct {
	betResultMap map[string]map[int]bool
	// map a CockID to a bet kind
	BetKindMap        map[string]com.BetKind
	PayoutMap         map[com.BetKind]com.Amount
	SmallLimitBetMap  map[com.Currency]map[com.BetKind]*com.BetLimit
	MediumLimitBetMap map[com.Currency]map[com.BetKind]*com.BetLimit
}

func (d *CockStrategyData) init() *CockStrategyData {
	d.initBetKindMap()
	d.PayoutMap = map[com.BetKind]com.Amount{}
	d.PayoutMap[BET_Straight] = 35 + 1

	// Load limit maps from JSON configuration
	d.loadLimitMapsFromJSON()
	return d
}

func (d *CockStrategyData) initBetKindMap() {
	d.betResultMap = map[string]map[int]bool{}
	d.BetKindMap = map[string]com.BetKind{}

	// Map CockIDs to BET_Straight
	d.BetKindMap[string(COCK_001)] = BET_Straight
	d.BetKindMap[string(COCK_002)] = BET_Straight
	d.BetKindMap[string(COCK_003)] = BET_Straight
	d.BetKindMap[string(COCK_004)] = BET_Straight
}

func (d *CockStrategyData) loadLimitMapsFromJSON() {
	configPath := "config/cock_strategy_limits.json"

	// Check if config file exists, if not use default values
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("CockStrategy config file %s not found, using default values\n", configPath)
		d.loadDefaultLimitMaps()
		return
	}

	// Read and parse JSON config
	jsonData, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading cockstrategy config file: %v, using default values\n", err)
		d.loadDefaultLimitMaps()
		return
	}

	var configData map[string]interface{}
	err = json.Unmarshal(jsonData, &configData)
	if err != nil {
		fmt.Printf("Error parsing cockstrategy config file: %v, using default values\n", err)
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

	fmt.Println("CockStrategy configuration loaded from JSON file")
}

func (d *CockStrategyData) loadDefaultLimitMaps() {
	d.SmallLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	usdc := map[com.BetKind]*com.BetLimit{}

	usdc[BET_Straight] = &com.BetLimit{Min: 0.1, Max: 3}
	d.SmallLimitBetMap[com.Currency("USDC")] = usdc

	d.MediumLimitBetMap = map[com.Currency]map[com.BetKind]*com.BetLimit{}
	usdc = map[com.BetKind]*com.BetLimit{}
	usdc[BET_Straight] = &com.BetLimit{Min: 1, Max: 30}
	d.MediumLimitBetMap[com.Currency("USDC")] = usdc
}
