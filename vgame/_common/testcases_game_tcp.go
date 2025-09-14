package com

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

type TestGameTcp struct {
	operatorId OperatorID
	Token      string
	vs         *VSocket

	gameDB   *sql.DB
	walletDB *sql.DB

	gameId     GameId
	roundId    RoundId
	gameNumber uint64
	payedout   bool
	resultNum  int

	roomList []*GameRoom
}

func (t *TestGameTcp) init() *TestGameTcp {
	t.operatorId = "001"
	t.vs = nil
	t.gameId = "001"
	t.roundId = 1
	t.resultNum = -1

	t.roomList = []*GameRoom{}

	return t
}

var TestGameTcpClient = (&TestGameTcp{}).init()

func (g *TestGameTcp) testCreateGameState() {
	db := g.gameDB
	// create a new game state and delete the old one
	tx, err := db.Begin()
	if err != nil {
		VUtils.PrintError(err)

		return
	}
	_, err2 := tx.Exec("DELETE FROM gamestate WHERE gameid=?", g.gameId)
	if err2 != nil {
		VUtils.PrintError(err2)
		return
	}

	//response, err3 := tx.Exec("INSERT INTO gamestate(gameid, roundid, state, payedout, statetime, result) VALUES(?,?,?,?,?,?)", g.gameId, g.roundId, GAME_STATE_STARTING, 0, 0, "")
	query := fmt.Sprintf("INSERT INTO gamestate(gameid, roundid, state, payedout, statetime, result) VALUES(%d,%d,%d,%d,%d,'%s')", g.gameId, g.roundId, GAME_STATE_STARTING, 0, 0, "")
	response, err3 := tx.Exec(query)
	if err3 != nil {
		tx.Rollback()
		VUtils.PrintError(err3)
		return
	}

	gameNumber, err1 := response.LastInsertId()
	if err1 != nil {
		tx.Rollback()
		VUtils.PrintError(err1)
		return
	}

	err4 := tx.Commit()
	if err4 != nil {
		VUtils.PrintError(err4)

		return
	}
	// assign new game number ---
	g.gameNumber = uint64(gameNumber)
	//fmt.Println("NEW GAME == gameId ", g.gameId, " gameNumber ", g.gameNumber)
}

func (g *TestGameTcp) saveGameState() {
	db := g.gameDB
	stateTime := float64(0)
	stateTime += 1
	stateTime += 1
	g.gameNumber = 5
	_, err := db.Exec("UPDATE gamestate SET state=?, statetime=? WHERE gamenumber=?", GAME_STATE_GEN_RESULT, stateTime, g.gameNumber)
	if err != nil {
		VUtils.PrintError(err)
		return
	}
}
func (g *TestGameTcp) genResult() {
	g.gameNumber = 5
	db := g.gameDB
	VUtils.RepeatCall(func(delay float64) {
		g.resultNum = VUtils.GetRandInt(36)
		_, err := db.Exec("UPDATE gamestate SET result=? WHERE gamenumber=?", strconv.Itoa(g.resultNum), g.gameNumber)
		if err != nil {
			VUtils.PrintError(err)
			return
		}
		//g.stateMng.nextState()
	}, 2.0, 1, nil)
}

func (g *TestGameTcp) saveResult(result string) {
	g.gameNumber = 5
	db := g.gameDB
	_, err := db.Exec("UPDATE gamestate SET result=? WHERE gamenumber=?", result, g.gameNumber)
	if err != nil {
		VUtils.PrintError(err)
		return
	}
}

func (g *TestGameTcp) GetGameInterface() bool {
	g.gameNumber = 5
	db := g.gameDB
	row := db.QueryRow("SELECT gamenumber, roundid, state, payedout, statetime, result FROM gamestate WHERE gameid=?", g.gameId)
	var currState = 0
	payedout := 0
	result := ""
	statetime := float64(0)
	err := row.Scan(&g.gameNumber, &g.roundId, &currState, &payedout, &statetime, &result)
	dur := 2
	fmt.Println("statetime >= 2 ", statetime >= float64(dur))
	if err != nil {
		fmt.Println("not existing previous state data for game ", g.gameId)
		return false
	}
	g.payedout = payedout > 0
	if result != "" {
		g.resultNum, err = strconv.Atoi(result)
		if err != nil {
			msg := fmt.Sprintf("can not parse game result for gameId %d and result %s ", g.gameId, result)
			VUtils.PrintError(errors.New(msg))
			return true
		}
		fmt.Println("loaded result ", g.resultNum)
	}

	// resume state
	//g.stateMng.setState(uint8(currState), statetime)
	return true
}

func (t *TestGameTcp) DoTest() {
	schemaName := fmt.Sprintf("%04d", t.operatorId)
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s", WALLET_MYSQL_USER, WALLET_MYSQL_KEY, WALLET_MYSQL_HOST, GAME_SCHEMA+"_"+schemaName)
	db, err := sql.Open("mysql", connStr)
	t.gameDB = db
	defer db.Close()

	if err != nil {
		fmt.Println(err)
		return
	}

	connStr = fmt.Sprintf("%s:%s@tcp(%s)/%s", WALLET_MYSQL_USER, WALLET_MYSQL_KEY, WALLET_MYSQL_HOST, WALLET_SCHEMA+"_"+schemaName)
	db2, err := sql.Open("mysql", connStr)
	t.walletDB = db2
	defer db2.Close()

	if err != nil {
		fmt.Println(err)
		return
	}

	//t.testCreateGameState()
	//t.saveGameState()
	//t.genResult()
	//time.Sleep(time.Second * 3)
	//t.saveResult("invalid_int")
	t.GetGameInterface()
	// wait for gen result

}
