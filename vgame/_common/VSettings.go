package com

import (
	"time"
)

type OperatorID string
type OperatorInfo struct {
	ID   OperatorID
	Name string
}

type VSettings struct {
	OPERATORS map[OperatorID]*OperatorInfo
}

func (setting *VSettings) init() *VSettings {
	setting.OPERATORS = map[OperatorID]*OperatorInfo{}
	setting.OPERATORS["001"] = &OperatorInfo{ID: "001", Name: "IGGame"}
	ROOM_ID_LENGTH = len(ROOM_ID_NONE)
	return setting
}

var GlobalSettings = (&VSettings{}).init()

// Runtime configurable variables (can be assigned from JSON)
var (
	WALLET_SCHEMA    = "vwallet"
	GAME_SCHEMA      = "vgame"
	WALLET_HOST      = "127.0.0.1" // for security always config it as a LAN ip
	WALLET_PORT      = "8092"
	WALLET_HTTP_PORT = "8099"
	PROXY_TCP_HOST   = "127.0.0.1" // for security always config it as a LAN ip
	PROXY_TCP_PORT   = "8093"
	PROXY_WSS_PORT   = "8094"
	BLOCKCHAIN_URL   = "http://localhost:8088/xrp"

	MAX_ACCOUNT          = 9999999
	MAX_PAYOUT_WAIT_TIME = 3.0

	WALLET_MYSQL_USER = "root"
	WALLET_MYSQL_KEY  = "hailuava12a6"
	WALLET_MYSQL_HOST = "127.0.0.1:3306"

	GAME_MYSQL_USER = "root"
	GAME_MYSQL_KEY  = "hailuava12a6"
	GAME_MYSQL_HOST = "127.0.0.1:3306"

	HASH_KEY             = "hailuava12a6"
	MAX_ADMIN_CONN       = 1000
	GAME_BATCH_MESSAGE   = false
	VSOCKET_SEND_TIMEOUT = 20 * time.Second

	// db
	WALLET_TABLE = "wallet"

	MAX_ROUND           = 9999
	ROOM_ID_LENGTH      = 9
	OPERATOR_ID_LENGTH  = 3
	MAX_TREND_PAGE_SIZE = 200
	HIS_PAGE_SIZE       = 20
)
