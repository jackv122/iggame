package com

type RoomManager struct {
	roomMap   map[string]*GameRoom
	RoomIdMap map[GameId][maxRoom]bool
}

// constructor
func (mng *RoomManager) Init() *RoomManager {
	// gg xxxx with gg is game id, xxxx is room id

	mng.roomMap = map[string]*GameRoom{}
	mng.RoomIdMap = map[GameId][maxRoom]bool{}

	// implement load from json config file ---

	RouletteIdPool := [maxRoom]bool{}
	GameAIdPool := [maxRoom]bool{}

	mng.RoomIdMap[IDRoulette] = RouletteIdPool
	mng.RoomIdMap[IDGameA] = GameAIdPool
	// ---

	return mng
}

func (mng *RoomManager) checkRoomExist(operatorId OperatorID, RoomId RoomId) bool {
	roomKey := getRoomKey(operatorId, RoomId)
	for key := range mng.roomMap {
		if roomKey == key {
			return true
		}
	}
	return false
}

func (mng *RoomManager) CreateRoom(GameId GameId, operatorId OperatorID, roomConf RoomConfig, server *GameServer) *GameRoom {

	RoomId := roomConf.RoomId
	gid := GameId
	if RoomId == "" {
		return nil
	}

	roomKey := getRoomKey(operatorId, RoomId)
	if mng.checkRoomExist(operatorId, RoomId) {
		return nil
	}

	room := (&GameRoom{GameId: gid, RoomId: RoomId}).Init(server, operatorId)
	room.roomConfig = &roomConf
	mng.roomMap[roomKey] = room

	game := GetGameInterface(gid, server)

	// init bet information for games
	game.InitRoomForGame(room)

	return room
}

func (mng *RoomManager) GetRoom(operatorId OperatorID, RoomId RoomId) *GameRoom {
	roomKey := getRoomKey(operatorId, RoomId)
	v, ok := mng.roomMap[roomKey]
	if !ok {
		return nil
	}
	return v
}
