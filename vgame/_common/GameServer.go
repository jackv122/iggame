package com

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type GameConfig struct {
	GameId      GameId
	FrameTime   float64
	OperatorIds []OperatorID
	RoomConfigs []RoomConfig
}

type ConnectionInfo struct {
	OperatorId OperatorID
	ConnId     ConnectionId
	UserId     UserId
	Currency   Currency
	RoomIdMap  map[RoomId]bool
	Mutex      sync.Mutex
}

func (info *ConnectionInfo) Init() *ConnectionInfo {
	info.Mutex = sync.Mutex{}
	info.RoomIdMap = map[RoomId]bool{}
	info.Currency = "USDC"
	return info
}

func (info *ConnectionInfo) joinRoom(roomId RoomId) {
	info.RoomIdMap[roomId] = true
}

func (info *ConnectionInfo) leaveRoom(roomId RoomId) {
	delete(info.RoomIdMap, roomId)
}

type ServerConfig struct {
	serverFrameTime float64
	workerNum       int
	GameConfigMap   map[GameId]*GameConfig
}

func (c *ServerConfig) Init() *ServerConfig {
	c.serverFrameTime = 1.0 / 5.0
	//c.serverFrameTime = 1.0
	c.workerNum = 4
	c.GameConfigMap = map[GameId]*GameConfig{}

	// Roulette ---------------
	RouletteConf := GameConfig{GameId: IDRoulette, FrameTime: 1.0}
	c.GameConfigMap[RouletteConf.GameId] = &RouletteConf
	RouletteConf.OperatorIds = []OperatorID{}
	for _, info := range GlobalSettings.OPERATORS {
		RouletteConf.OperatorIds = append(RouletteConf.OperatorIds, info.ID)
	}
	RouletteConf.RoomConfigs = []RoomConfig{{RoomId: "001001", limitLevel: LIMIT_LEVEL_LOW}, {RoomId: "001002", limitLevel: LIMIT_LEVEL_NORMAL}}
	/*
		// GameA ---------------
		GameAConf := GameConfig{GameId: IDGameA, FrameTime: 1.0}
		c.GameConfigMap[GameAConf.GameId] = &GameAConf
		GameAConf.OperatorIds = []OperatorID{}
		for _, info := range GlobalSettings.OPERATORS {
			GameAConf.OperatorIds = append(RouletteConf.OperatorIds, info.ID)
		}
		GameAConf.RoomConfigs = []RoomConfig{{roomId: "002001", limitLevel: LIMIT_LEVEL_LOW}, {roomId: "002002", limitLevel: LIMIT_LEVEL_NORMAL}}
	*/

	return c
}

type Task struct {
	taskId int

	ConnId     ConnectionId
	OperatorId OperatorID
	roomId     RoomId

	params []unsafe.Pointer
}

func (t *Task) Init() *Task {
	t.params = []unsafe.Pointer{}
	return t
}

var GameServerConfig = (&ServerConfig{}).Init()

type GameServer struct {
	proxyConn  *VSocket
	WalletConn *VSocket

	DB       *sql.DB
	taskList chan *Task
	RoomMng  *RoomManager
	connMap  map[ConnectionId]*ConnectionInfo

	// TODO: should map [GameId_roomId] to a game instance since:
	// 		1 room can bind with 1 game instance (tienlen 1-1) or n room can bind with 1 game instance (roulette n-1)
	gameMap          map[GameId]unsafe.Pointer
	timeKeeper       *TimeKeeper
	connMessageMutex sync.Mutex
	connMapMutex     sync.Mutex
	connMessageMap   map[ConnectionId]([]*string)
	isMaintenance    bool

	isStarted bool
}

func (server *GameServer) Init() *GameServer {
	server.taskList = make(chan *Task)
	server.RoomMng = (&RoomManager{}).Init()
	server.connMap = map[ConnectionId]*ConnectionInfo{}
	server.gameMap = map[GameId]unsafe.Pointer{}
	server.connMessageMap = map[ConnectionId]([]*string){}
	server.timeKeeper = &TimeKeeper{}
	server.connMessageMutex = sync.Mutex{}
	server.connMapMutex = sync.Mutex{}

	return server
}

var Game = (&GameServer{}).Init()

func (server *GameServer) SetGamePointer(gameId GameId, p unsafe.Pointer) {
	server.gameMap[gameId] = p
}

func (server *GameServer) GetGamePointer(gameId GameId) unsafe.Pointer {
	return server.gameMap[gameId]
}

func (server *GameServer) performTask(task *Task) {
	ConnId := task.ConnId
	connInfo, ok := server.GetConnectionInfo(ConnId)
	if !ok {
		return
	}

	OperatorId := task.OperatorId
	roomId := task.roomId

	room := server.RoomMng.GetRoom(OperatorId, roomId)
	game := GetGameInterface(room.GameId, server)

	switch task.taskId {

	case TASK_PROCESS_MSG:
		for _, param := range task.params {
			msg := (*string)(param)
			connInfo.Mutex.Lock()
			game.OnMessage(ConnId, *msg)
			connInfo.Mutex.Unlock()
		}
	}
}

func (server *GameServer) registerConnectionInfo(info *ConnectionInfo) {
	server.connMapMutex.Lock()
	server.connMap[info.ConnId] = info
	server.connMapMutex.Unlock()
}

func (server *GameServer) GetConnectionInfo(ConnId ConnectionId) (*ConnectionInfo, bool) {
	server.connMapMutex.Lock()
	defer server.connMapMutex.Unlock()
	info, ok := server.connMap[ConnId]
	return info, ok
}

func (server *GameServer) removeConnectionInfo(ConnId ConnectionId) {
	server.connMapMutex.Lock()
	delete(server.connMap, ConnId)
	server.connMapMutex.Unlock()
}

// global functions --------------------------------------

func (server *GameServer) SendPublicMessage(ConnIds []ConnectionId, data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		VUtils.PrintError(err)
	}
	msg := ProxyBroadcastMessage{}
	msg.CMD = PROXY_CMD_BROADCAST
	msg.ConnIds = ConnIds
	//msg.Data = b64.URLEncoding.EncodeToString(bytes)
	msg.Data = string(bytes)

	content, err := json.Marshal(msg)
	if err != nil {
		VUtils.PrintError(err)
		return
	}

	server.proxyConn.Send(content, nil, nil)
}

func (server *GameServer) SendPrivateMessage(ConnId ConnectionId, data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		VUtils.PrintError(err)
	}
	msg := ProxyBroadcastMessage{}
	msg.CMD = PROXY_CMD_BROADCAST
	msg.ConnIds = []ConnectionId{ConnId}
	//msg.Data = b64.URLEncoding.EncodeToString(bytes)
	msg.Data = string(bytes)

	content, err := json.Marshal(msg)
	if err != nil {
		VUtils.PrintError(err)
		return
	}

	server.proxyConn.Send(content, nil, nil)
}

func (server *GameServer) DisconnectClient(ConnId ConnectionId, reason string) {
	// TODO: send message to client the reason
	// TODO: send private message to proxy to close the connection
	msg := ProxyBroadcastMessage{}
	msg.CMD = PROXY_CMD_DISCONNECT_CLIENT
	msg.ConnIds = []ConnectionId{ConnId}
	msg.Data = ""

	content, err := json.Marshal(msg)
	if err != nil {
		VUtils.PrintError(err)
		return
	}

	server.proxyConn.Send(content, nil, nil)
	server.OnClientDisconnect(ConnId)
}

func (server *GameServer) OnClientConnect(OperatorId OperatorID, ConnId ConnectionId, UserId UserId) {
	connInfo := (&ConnectionInfo{ConnId: ConnId, OperatorId: OperatorId, UserId: UserId}).Init()
	server.registerConnectionInfo(connInfo)
}

func (server *GameServer) OnClientDisconnect(ConnId ConnectionId) {
	conInfo, ok := server.GetConnectionInfo(ConnId)
	if !ok {
		return
	}
	for roomId, _ := range conInfo.RoomIdMap {
		room := server.RoomMng.GetRoom(conInfo.OperatorId, roomId)
		if room != nil {
			room.LeaveRoom(ConnId)
		}
	}
	server.removeConnectionInfo(ConnId)
}

func (server *GameServer) OnClientMessage(ConnId ConnectionId, data []byte) {

	connInfo, ok := server.GetConnectionInfo(ConnId)
	if !ok {
		return
	}

	roomId := RoomId(data[:ROOM_ID_LENGTH])
	_, isJoinedRoom := connInfo.RoomIdMap[roomId]
	msg := string(data[ROOM_ID_LENGTH:])

	// all clients MUST join a room success before can send any other message
	// process JOIN_ROOM and LEAVE_ROOM message sequencially in network thread
	if !isJoinedRoom {
		var obj map[string]interface{}
		err := json.Unmarshal([]byte(msg), &obj)
		if err != nil {
			VUtils.PrintError(err)
			return
		}
		cmd := obj["CMD"].(string)
		if cmd != CMD_JOIN_ROOM {
			return
		}

		// check valid roomId
		if !server.RoomMng.checkRoomExist(connInfo.OperatorId, roomId) {
			return
		}

		room := server.RoomMng.GetRoom(connInfo.OperatorId, roomId)

		existing, orgConnId := room.checkUserExist(connInfo.UserId)

		// 2 connections join the same room => kick the old one
		if existing {
			//DisconnectClient(orgConnId, "duplicate join room")
			room.LeaveRoom(orgConnId)
		}

		room.JoinRoom(ConnId)

	} else {

		if GAME_BATCH_MESSAGE {
			// batch all messages for the room ---
			// this way will reduce number of TASK_PROCESS_MSG push to taskList => reduce distributing the same room tasks to many threads
			server.connMessageMutex.Lock()
			defer server.connMessageMutex.Unlock()
			messages, ok := server.connMessageMap[ConnId]
			if !ok {
				messages = []*string{}
			}

			server.connMessageMap[ConnId] = append(messages, &msg)
			// -----
		} else {
			params := []unsafe.Pointer{unsafe.Pointer(&msg)}
			task := Task{ConnId: ConnId, OperatorId: connInfo.OperatorId, roomId: roomId, taskId: TASK_PROCESS_MSG, params: params}
			//server.taskList <- &task
			server.performTask(&task)
		}
	}

}

// for game batching message
func (server *GameServer) startServerUpdate() {
	// start server update
	update := func(dt float64) {
		// check message
		server.connMessageMutex.Lock()
		defer server.connMessageMutex.Unlock()
		for ConnId, messages := range server.connMessageMap {
			params := []unsafe.Pointer{}
			for _, msg := range messages {
				params = append(params, unsafe.Pointer(msg))
			}

			task := Task{ConnId: ConnId, taskId: TASK_PROCESS_MSG, params: params}
			server.taskList <- &task
		}
		// reset roomMessageMap
		server.connMessageMap = map[ConnectionId][]*string{}
	}

	VUtils.RepeatCall(update, GameServerConfig.serverFrameTime, 0, server.timeKeeper)
}

func (s *GameServer) testVsocket() {
	for i := 0; i < 10; i++ {
		go func() {
			strParam := ClientStringGameResponse{}
			strParam.Str = "hi from GameServer 1" + strconv.Itoa(i)
			s.SendPublicMessage([]ConnectionId{}, strParam)
			strParam = ClientStringGameResponse{}
			strParam.Str = "hi from GameServer 2" + strconv.Itoa(i)
			s.SendPublicMessage([]ConnectionId{}, strParam)
			strParam = ClientStringGameResponse{}
			strParam.Str = "hi from GameServer 3" + strconv.Itoa(i)
			s.SendPublicMessage([]ConnectionId{}, strParam)
		}()
	}

}

func (s *GameServer) Start() {

	// init local conns ---
	// wallet conn
	conn, err := net.Dial("tcp", WALLET_HOST+":"+WALLET_PORT)
	if err != nil {
		panic(err)
	}
	s.WalletConn = (&VSocket{}).Init(conn, s.onWalletMessageHdl, s.onWalletCloseHdl, true, 4, GAME_ENCRYPT)
	serverType := uint8(LOCAL_CONN_TYPE_GAME)
	bytes := []byte{serverType}
	cmd := uint16(WCMD_REGISTER_CONN)
	pres := VUtils.Uint16ToBytes(cmd)
	bytes = append(pres, bytes...)
	// dont need OperatorId for cmd WCMD_REGISTER_CONN
	s.WalletConn.Send(bytes, nil, nil)

	// proxy conn
	conn, err = net.Dial("tcp", PROXY_TCP_HOST+":"+PROXY_TCP_PORT)
	if err != nil {
		panic(err)
	}
	s.proxyConn = (&VSocket{}).Init(conn, s.onProxyMessageHdl, s.onProxyCloseHdl, true, 4, PROXY_ENCRYPT)
	//s.testVsocket()
	// ----

	s.isStarted = true
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s", GAME_MYSQL_USER, GAME_MYSQL_KEY, GAME_MYSQL_HOST, GAME_SCHEMA)
	db, err := sql.Open("mysql", connStr)
	s.DB = db

	if err != nil {
		fmt.Println(err)
		return
	}

	if GAME_BATCH_MESSAGE {
		s.startServerUpdate()
	}

	// init worker threads ----
	// Note: dont create thread here. The threads is generated by vsocket already ---
	/*
		for i := 0; i < GameServerConfig.workerNum; i++ {
			go func() {
				for task := range s.taskList {
					s.performTask(task)
				}
			}()
		}
	*/
	// ------------------------------------------------------------------------
	// init games ---
	limitMap := map[GameId]([](map[Currency]map[BetKind]*BetLimit)){}
	roomIdMap := map[GameId][]RoomId{}
	for _, gameConf := range GameServerConfig.GameConfigMap {

		GameFactory(gameConf.GameId, s)

		game := GetGameInterface(gameConf.GameId, s)
		limitMap[gameConf.GameId] = game.GetAllBetLimits()

		roomIds := []RoomId{}
		for _, roomConf := range gameConf.RoomConfigs {
			roomIds = append(roomIds, roomConf.RoomId)
		}
		roomIdMap[gameConf.GameId] = roomIds
	}

	// init rooms ---
	for _, gameConf := range GameServerConfig.GameConfigMap {
		game := GetGameInterface(gameConf.GameId, s)
		for _, OperatorId := range gameConf.OperatorIds {
			for _, roomConf := range gameConf.RoomConfigs {
				roomConf.limitBetMap = game.GetBetLimit(roomConf.limitLevel)
				room := s.RoomMng.CreateRoom(gameConf.GameId, OperatorId, roomConf, s)
				if room == nil {
					continue
				}
			}
		}
	}

	msg := ProxyRegisterRoomMessage{}
	msg.CMD = PROXY_CMD_REGISTER_ROOMS
	msg.RoomIdMap = roomIdMap
	msg.LimitMap = &limitMap

	content, _ := json.Marshal(msg)
	s.proxyConn.Send(content, nil, nil)

	fmt.Println("server started.")

	// make sure all DB and server initialize successfully before start process client messages

	for GameId := range s.gameMap {
		game := GetGameInterface(GameId, s)
		game.Start()
	}
}

func (server *GameServer) Stop() {
	if !server.isStarted {
		return
	}
	server.isStarted = false
	for GameId := range server.gameMap {
		game := GetGameInterface(GameId, server)
		game.Stop()
	}
	server.DB.Close()
	fmt.Println("all game operators stopped.")
}

func (server *GameServer) getAllUserConns() []ConnectionId {
	ConnIds := []ConnectionId{}
	for ConnId := range server.connMap {
		ConnIds = append(ConnIds, ConnId)
	}
	return ConnIds
}

func (server *GameServer) Maintenance() {
	fmt.Println("server game maintenance")
	if server.isMaintenance {
		return
	}

	server.isMaintenance = true
	res := ClientNumberGameResponse{}
	res.CMD = CMD_MAINTENANCE
	ConnIds := server.getAllUserConns()
	server.SendPublicMessage(ConnIds, res)

	time.Sleep(time.Second * 2)

	// only save schedule actions
	for GameId := range server.gameMap {
		game := GetGameInterface(GameId, server)
		game.SaveGameState()
	}
	server.Stop()
}

func (s *GameServer) onWalletMessageHdl(vs *VSocket, requestId uint64, data []byte) {
	cmd := VUtils.BytesToUint16(&data)
	//body := data[2:]
	switch cmd {
	case WCMD_MAINTENANCE:
		s.Maintenance()
	}
}

func (s *GameServer) onWalletCloseHdl(vs *VSocket) {
	s.Maintenance()
}

// parsing proxy message
func (s *GameServer) onProxyMessageHdl(vs *VSocket, requestId uint64, dataBytes []byte) {

	data := ProxyClientMessage{}
	err := json.Unmarshal(dataBytes, &data)
	if err != nil {
		fmt.Println("dataStr ", string(dataBytes))
		VUtils.PrintError(err)
		s.Maintenance()
		return
	}

	switch data.CMD {
	case PROXY_CMD_CLIENT_CONNECT:
		s.OnClientConnect(data.OperatorId, data.ConnId, data.UserId)
	case PROXY_CMD_CLIENT_DISCONNECT:
		s.OnClientDisconnect(data.ConnId)
	case PROXY_CMD_CLIENT_MSG:
		s.OnClientMessage(data.ConnId, []byte(data.Data)) // TODO: check if need to decode Base64 data.Data
	}

}

func (s *GameServer) onProxyCloseHdl(vs *VSocket) {
	s.Maintenance()
}

func (s *GameServer) GetGameConf(GameId GameId) *GameConfig {
	return GameServerConfig.GameConfigMap[GameId]
}

func (s *GameServer) SaveGameResult(gameNumber GameNumber, GameId GameId, roundId RoundId, currState GameState, stateTime float64, resultStr string, txh string, w string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		VUtils.PrintError(err)
		return err
	}

	str := fmt.Sprintf("%d_%s_%d_%d_%s_%s_%s", gameNumber, GameId, roundId, currState, resultStr, txh, w)
	hash := VUtils.HashString(str)
	_, err2 := tx.Exec("UPDATE gamestate SET state=?, statetime=?, result=?, tx=?, w=?, h=? WHERE gamenumber=?", currState, stateTime, resultStr, txh, w, hash, gameNumber)
	if err2 != nil {
		tx.Rollback()
		VUtils.PrintError(err2)
		s.Maintenance()
		return err2

	}

	str2 := fmt.Sprintf("%d_%s_%d_%s_%s_%s", gameNumber, GameId, roundId, resultStr, txh, w)
	h := VUtils.HashString(str2)

	_, err3 := tx.Exec("INSERT INTO trend(gamenumber, gameid, roundid, result, tx, w, h) VALUES(?,?,?,?,?,?,?)", gameNumber, GameId, roundId, resultStr, txh, w, h)

	if err3 != nil {
		tx.Rollback()
		VUtils.PrintError(err3)
		s.Maintenance()
		return err3
	}
	err4 := tx.Commit()
	if err4 != nil {
		VUtils.PrintError(err4)
		s.Maintenance()
		return err4
	}
	return nil
}

func (s *GameServer) LoadTrends(GameId GameId, page uint32) []*TrendItem {
	startRow := page * TREND_PAGE_SIZE
	endRow := (page + 1) * TREND_PAGE_SIZE
	query := fmt.Sprintf("SELECT gamenumber, roundid, result, tx, w FROM trend WHERE gameid='%s' AND result>'' ORDER BY updatetime DESC LIMIT %d, %d", GameId, startRow, endRow)
	//fmt.Println("query == ", query)
	rows, err := s.DB.Query(query)

	if err != nil {
		VUtils.PrintError(err)
		return nil
	}
	defer rows.Close()
	trends := []*TrendItem{}
	for rows.Next() {
		trend := TrendItem{}
		err := rows.Scan(&trend.GameNumber, &trend.RoundId, &trend.Result, &trend.Txh, &trend.W)
		if err != nil {
			VUtils.PrintError(err)
			s.Maintenance()
			return nil
		}
		trends = append(trends, &trend)
	}
	return trends
}
