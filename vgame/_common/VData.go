package com

type UserId uint64
type ConnectionId uint64
type Amount float64
type BetType string
type GameId string
type GameNumber uint64
type BettingId uint64
type RoomId string
type RoundId uint32
type SeatId uint8
type BetKind uint8
type Currency string
type LimitLevel uint
type EncryptType uint8
type GameState uint8

type TrendItem struct {
	GameNumber GameNumber
	RoundId    RoundId
	Result     string
	Txh        string
	W          string
}

type BlockChainTxResult struct {
	ErrorCode    int
	ErrorMessage string
	W            string
	Txh          string
}

const (
	IDRoulette     GameId = "Roulette_01"
	IDCockStrategy GameId = "CockStrategy_01"
	IDGameA        GameId = "GameA_01"

	GAME_ENCRYPT      = 0
	PROXY_ENCRYPT     = 1
	maxRoom       int = 100
)

const (
	LIMIT_LEVEL_SMALL  = 0
	LIMIT_LEVEL_MEDIUM = 1
)

type SecureInt struct {
}

const (
	LOCAL_CONN_TYPE_PROXY = 0
	LOCAL_CONN_TYPE_GAME  = 1
)

type LocalConnInfo struct {
	ConnType uint8
}

const (
	// proxy cmds
	PCMD_REGISTER_OPERATORS = 0

	// wallet cmds
	WCMD_REGISTER_CONN = 0
	WCMD_MAINTENANCE   = 1

	WCMD_GET_BALANCE = 2
	WCMD_DEPOSIT     = 3
	WCMD_WIDTHDRAW   = 4
	WCMD_CREATE_ACC  = 5

	WCMD_GET_BALANCE_LIST      = 6
	WCMD_ADD_BALANCE_LIST      = 7
	WCMD_SUBTRACT_BALANCE_LIST = 8

	// schedule write db => alwayse use transaction to update (balance + all bettings of the userid)
	// deposit / withdraw => sync tx write betting & balance immediately => return success if the tx complete else REJECT deposit / withdraw action
	WCMD_SAVE_BETTING  = 9
	WCMD_CLEAR_BETTING = 10 // for old game number
	WCMD_QUERY_BETTING = 11

	WCMD_HISTORY = 12
)

type AmountInfo struct {
	UserId UserId
	Amount Amount
}

type BalanceInfo struct {
	UserId   UserId
	Amount   Amount
	Currency Currency
}

const (
	// proxy command
	PROXY_CMD_BROADCAST         = 0
	PROXY_CMD_CLIENT_CONNECT    = 1
	PROXY_CMD_CLIENT_DISCONNECT = 2
	PROXY_CMD_CLIENT_MSG        = 3

	PROXY_CMD_REGISTER_ROOMS    = 4
	PROXY_CMD_DISCONNECT_CLIENT = 5

	// task
	TASK_PROCESS_MSG = 0
)

type ProxyClientMessage struct {
	CMD        int
	OperatorId OperatorID
	ConnId     ConnectionId // use incase broadcast to client
	UserId     UserId
	Data       string
}

type ProxyRegisterRoomMessage struct {
	CMD       int
	RoomIdMap map[GameId][]RoomId // use incase broadcast to client
	LimitMap  *map[GameId]([](map[Currency]map[BetKind]*BetLimit))
}

type ProxyBroadcastMessage struct {
	CMD     int
	ConnIds []ConnectionId // use incase broadcast to client
	Data    string
}

type ProxyMessage struct {
	CMD    int
	ConnId ConnectionId
	Data   string
}

const (
	CMD_JOIN_ROOM          = "joinroom"
	CMD_LEAVE_ROOM         = "leaveroom"
	CMD_LEAVE_ROOM_SUCCESS = "leaveroomsuccess"

	CMD_AUTH         = "auth"
	CMD_AUTH_SUCCESS = "authplayersucceed"
	CMD_AUTH_EXPIRED = "authplayerexpired"
	CMD_DOUBLE_LOGIN = "doublelogin"

	CMD_OPERATOR_FLAG = "operatorflag" // {"operator":"W88","miniStatus":1,"tipStatus":1}}

	CMD_BALANCE     = "balance"
	CMD_SERVER_TIME = "servertime"
	//{"kind":"showlobby","content":[{"id":1,"gameName":"C baccarat","dealerName":"Girlie","ticks":40,"status":1,"dealerPhoto":"http://casino.w88bet.com/imgs/lucia.jpg","dealerTableId":1,"dealerGameName":"baccarat","displayStatus":1}
	CMD_SHOW_LOBBY = "showlobby"

	CMD_GET_BALANCE       = "getbalance"
	CMD_MINI_BET_STATS    = "minibetstats"
	CMD_LIVE_BET_STATS    = "livebetstats"
	CMD_USER_BET_STATS    = "userbetstats"
	CMD_BUNDLE_RESULT     = "bundleresult"
	CMD_GAME_RESULT       = "gameresult"
	CMD_LIMITS            = "limits"
	CMD_ROLL_BACK_SUCCEED = "rollbacksucceed"

	CMD_ENTER = "enter"

	//{"kind":"joinroomfailed","content":{"table":101,"error":-20001}}
	CMD_JOIN_ROOM_FAIL = "joinroomfailed"

	//{"kind":"roomlist","table":101}
	//{"kind":"roomlist","content":{"table":101,"rooms":[[2,0],[3,0],[4,0],[5,0],[6,0],[8,0],[9,0],[10,0],[1,1],[7,1]]}}
	CMD_ROOM_LIST   = "roomlist"
	CMD_TABLE_LIMIT = "tablelimit"
	CMD_GET_TRENDS  = "gettrends"
	CMD_TRENDS      = "trends"
	CMD_GET_HISTORY = "gethistory"
	CMD_HISTORY     = "history"
	CMD_LAST_ROUND  = "lastround"

	CMD_TICK = "tick"

	CMD_CARDS     = "cards"
	CMD_DEALER    = "dealer" // has dealer or not
	CMD_BET_LIMIT = "betlimit"

	CMD_JOIN_GAME         = "joingame"
	CMD_JOIN_GAME_SUCCESS = "joingamesuccess"

	CMD_JOIN_ROOM_SUCCESS = "joinroomsuccess"
	CMD_ROOM_INFO         = "roominfo"
	CMD_GET_ROOM_INFO     = "getroominfo"
	CMD_ROOM_UPDATE       = "roomupdate"

	CMD_START_GAME        = "startgame"
	CMD_START_BET_SUCCEED = "startbetsucceed"
	CMD_COUNT_DOWN        = "countdown"
	CMD_STOP_BET_SUCCEED  = "stopbetsucceed"
	CMD_END_BET_SUCCEED   = "endbetsucceed"

	CMD_POOL               = "pool"
	CMD_DEAL               = "deal"
	CMD_SEND_BET_UPDATE    = "betupdate"
	CMD_BET_UPDATE_SUCCEED = "betupdatesucceed"
	CMD_BET_UPDATE_FAIL    = "betupdatefail"

	CMD_COMMIT_SUCCEED   = "commitsucceed"
	CMD_REWARD           = "reward"
	CMD_NEW_SHOE_SUCCEED = "newshoesucceed"
	CMD_HISTORY_100      = "history100"
	CMD_DEALER_CHANGED   = "dealerchanged"
	CMD_DEALER_LEAVE     = "dealerleave"
	CMD_NOTICE           = "notice"

	CMD_PAYOUT_SUCCESS = "payoutsuccess"

	CMD_LOGIN_DUPLICATED = "loginduplicated"
	CMD_EXIT             = "exit"
	//ws.onmessage: {"kind":"roombetsucceed","content":{"table":101,"room":1,"seat":7,"nickname":"vuanh","cats":["11","9","1","2","3"],"amounts":[5,5,5,5,10]}}
	CMD_ROOM_BET_SUCCEED = "roombetsucceed"
	CMD_UPDATE_AVATAR    = "updateavatar"
	//{"kind":"updateavatar","id":"14"}
	//{"kind":"updateAvatarSucceed","content":"14"}
	CMD_UPDATE_AVATAR_SUCCEED = "updateAvatarSucceed"
	CMD_BET_HISTORY           = "bethistory"
	//{"table":103,"amount":1,"from":"2d","kind":"tip"}
	// {"kind":"tipsucceed","content":{"table":103,"value":1,"time":"None"}}
	CMD_TIP               = "tip"
	CMD_TIP_SUCCEED       = "tipsucceed"
	CMD_VOID_HAND_SUCCESS = "voidhandsucceed"
	CMD_MAINTENANCE       = "maintenance"
)

const (
	RES_FAIL_BET_REJECT       = 0
	RES_FAIL_BET_MIN          = 1
	RES_FAIL_BET_MAX          = 2
	RES_FAIL_BET_INSUFFICIENT = 3
)

const (
	WERR_INVALID_PARAM_FORMAT = 0
	WERR_INVALID_PARAM_JSON   = 1
	WERR_USER_NOT_EXIST       = 2
	WERR_INSUFFICIENT_BALANCE = 30
	WERR_CREATE_BETTING_ERROR = 4
	WERR_WRITE_BETTING_ERROR  = 5
)

const (
	GAME_STATE_STARTING      = 0
	GAME_STATE_BETTING       = 1
	GAME_STATE_CLOSE_BETTING = 2
	GAME_STATE_GEN_RESULT    = 3
	GAME_STATE_RESULT        = 4
	GAME_STATE_PAYOUT        = 5
)

const (
	BET_COMMON = 0
)

// BetFailResponse -----------
// /{CMD: GameCMD.BET_UPDATE_FAIL, Content: {Balance: 100.0, BetTypes:[], BetAmonts:[]} }

type BetFailResponse struct {
	CMD        string
	GameId     GameId
	RoomId     RoomId
	RoundId    RoundId
	FailCode   int
	Balance    Amount
	BetTypes   []BetType
	BetAmounts []Amount
}

func (res *BetFailResponse) Init(room *GameRoom) *BetFailResponse {
	res.CMD = CMD_BET_UPDATE_FAIL

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.RoundId = game.GetRoundId()

	res.FailCode = 0
	return res
}

// BaseGameResponse -----------

type BaseGameResponse struct {
	CMD        string
	GameId     GameId
	RoomId     RoomId
	GameNumber GameNumber
	RoundId    RoundId
}

func (res *BaseGameResponse) Init(room *GameRoom, cmd string) *BaseGameResponse {
	res.CMD = cmd

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.GameNumber = game.GetGameNumber()
	res.RoundId = game.GetRoundId()
	return res
}

// TickResponse -----------

type TickResponse struct {
	CMD          string
	GameId       GameId
	RoomId       RoomId
	RoundId      RoundId
	Time         uint16
	RoomTotalBet float64
}

func (res *TickResponse) Init(room *GameRoom) *TickResponse {
	res.CMD = CMD_TICK

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.RoundId = game.GetRoundId()
	return res
}

// EndBettingResponse -----------

type EndBetResponse struct {
	CMD      string
	GameId   GameId
	RoomId   RoomId
	RoundId  RoundId
	BetState []*BetPlace
	Balance  Amount
}

func (res *EndBetResponse) Init(room *GameRoom) *EndBetResponse {
	res.CMD = CMD_END_BET_SUCCEED

	res.GameId = room.GameId
	res.RoomId = room.RoomId
	game := GetGameInterface(room.GameId, room.Server)
	res.RoundId = game.GetRoundId()
	return res
}
