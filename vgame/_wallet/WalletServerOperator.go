package wal

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	com "vgame/_common"

	_ "github.com/go-sql-driver/mysql"
)

// tcp message: [msglen - 2bytes][ck - 8bytes][isResponse - 1byte][requestId - 8bytes][cmd - 2bytes][json]

type WalletServerOperator struct {
	operatorId      com.OperatorID
	userMap         map[com.UserId]*com.UserWallet
	userMapMutex    sync.Mutex
	bettingMap      map[com.BettingId]*com.BettingRecord
	bettingMapMutex sync.Mutex
	db              *sql.DB
	createAccMutex  sync.Mutex
	isMaintenance   bool
}

func (s *WalletServerOperator) Init() *WalletServerOperator {
	s.userMap = map[com.UserId]*com.UserWallet{}
	s.bettingMap = map[com.BettingId]*com.BettingRecord{}

	return s
}

func (s *WalletServerOperator) Start(operatorId com.OperatorID) {
	//hash := Hashcom.Amount(2, 12)
	//fmt.Println("md5 hash len ", hash)
	s.operatorId = operatorId
	schemaName := com.WALLET_SCHEMA + "_" + string(operatorId)
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", com.WALLET_MYSQL_USER, com.WALLET_MYSQL_KEY, com.WALLET_MYSQL_HOST, schemaName)
	fmt.Println("Wallet start schemaName ", schemaName)
	db, err := sql.Open("mysql", connStr)
	s.db = db

	if err != nil {
		fmt.Println(err)
		return
	}

	var version string

	err2 := db.QueryRow("SELECT VERSION()").Scan(&version)

	if err2 != nil {
		fmt.Println(err2)
	}

	fmt.Println("mysql version ", version)

	// load DB
	if !s.LoadDB() {
		fmt.Println(errors.New("wrong hash from DB"))
		return
	}

	// Schedule write DB
	com.VUtils.RepeatCall(s.scheduleWriteDB, 5.0, 0, &com.TimeKeeper{})
}

func (s *WalletServerOperator) onProxyMsgHdl(vs *com.VSocket, requestId uint64, data []byte) {
	cmd := com.VUtils.BytesToUint16(&data)
	body := data[2:]
	switch cmd {
	case com.WCMD_GET_BALANCE:
		s.getBalanceTcp(vs, requestId, &body)
	case com.WCMD_HISTORY:
		s.loadHistoryTcp(vs, requestId, &body)
	}
}

// [cmd - 2bytes][json]
func (s *WalletServerOperator) onGameMsgHdl(vs *com.VSocket, requestId uint64, data []byte) {
	cmd := com.VUtils.BytesToUint16(&data)
	body := data[2:]
	switch cmd {
	case com.WCMD_CREATE_ACC:
		s.createaccTcp(vs, requestId, &body)

	case com.WCMD_DEPOSIT:
		s.depositTcp(vs, requestId, &body)
	case com.WCMD_WIDTHDRAW:
		s.widthdrawTcp(vs, requestId, &body)
	case com.WCMD_GET_BALANCE:
		s.getBalanceTcp(vs, requestId, &body)

	case com.WCMD_ADD_BALANCE_LIST:
		s.addBalanceListTcp(vs, requestId, &body)
	case com.WCMD_SUBTRACT_BALANCE_LIST:
		s.subtractBalanceListTcp(vs, requestId, &body)
	case com.WCMD_GET_BALANCE_LIST:
		s.getBalanceListTcp(vs, requestId, &body)

	case com.WCMD_SAVE_BETTING:
		s.saveBettingTcp(vs, requestId, &body)
	case com.WCMD_CLEAR_BETTING:
		s.clearBettingTcp(vs, requestId, &body)
	case com.WCMD_QUERY_BETTING:
		s.queryBettingTcp(vs, requestId, &body)
	}
}

func (t *WalletServerOperator) onCloseHdl(vs *com.VSocket) {

}

func (s *WalletServerOperator) LoadDB() bool {
	db := s.db
	// Execute the query
	results, err := db.Query("SELECT userid, balance, currency, h FROM wallet")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer results.Close()

	for results.Next() {
		userWallet := (&com.UserWallet{}).Init()
		err = results.Scan(&userWallet.UserId, &userWallet.Balance, &userWallet.Currency, &userWallet.Hash)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}
		hash := com.VUtils.HashAmount(uint64(userWallet.UserId), userWallet.Balance, userWallet.Currency)
		if hash != userWallet.Hash {
			fmt.Println("wrong hash: ", userWallet.UserId, hash, userWallet.Hash)
			return false
		}
		s.userMap[userWallet.UserId] = userWallet
	}
	fmt.Println("load DB success")
	return true
}

func (s *WalletServerOperator) findUid() com.UserId {
	for i := com.UserId(1); i < com.UserId(com.MAX_ACCOUNT); i++ {
		_, ok := s.userMap[i]
		if !ok {
			return i
		}
	}
	return 0
}

func (s *WalletServerOperator) createaccTcp(vs *com.VSocket, requestId uint64, body *[]byte) {
	response := (&com.CreateAccResponse{}).Init()
	defer func() {
		res, _ := json.Marshal(response)
		vs.Response(requestId, res)
	}()
	currency := com.Currency(*body)
	//fmt.Println("createaccTcp currency ", currency)
	err := s.createacc(response, currency)
	if err != nil {
		com.VUtils.PrintError(err)
	}
}

func (s *WalletServerOperator) createacc(res *com.CreateAccResponse, currency com.Currency) error {
	s.createAccMutex.Lock()
	defer s.createAccMutex.Unlock()
	db := s.db
	amt := com.Amount(0)
	uid := s.findUid()
	//fmt.Println("createacc uid === ", uid)

	if uid == 0 {
		com.VUtils.PrintError(errors.New("no more uid"))
		res.ErrorCode = 0
		return errors.New("")
	}
	hash := com.VUtils.HashAmount(uint64(uid), amt, currency)
	query := "INSERT INTO wallet (userid, balance, currency, h) VALUES ( ?, ?, ?, ? )"
	_, err := db.Exec(query, uid, amt, currency, hash)
	if err != nil {
		com.VUtils.PrintError(errors.New("sql error: " + query))
		res.ErrorCode = 1
		return err
	}
	// update datas
	userWallet := (&com.UserWallet{}).Init()
	userWallet.UserId = uid
	userWallet.Balance = amt
	userWallet.Currency = currency
	userWallet.Hash = com.VUtils.HashAmount(uint64(uid), amt, currency)
	s.userMapMutex.Lock()
	s.userMap[uid] = userWallet
	s.userMapMutex.Unlock()

	// response

	res.UserId = uid

	return nil
}

// API for list users ------------

func (s *WalletServerOperator) getBalanceListTcp(vs *com.VSocket, requestId uint64, body *[]byte) {

	response := (&com.BalanceListResponse{}).Init()
	defer func() {
		res, _ := json.Marshal(response)
		vs.Response(requestId, res)
	}()

	ul := []com.UserId{}
	err := json.Unmarshal(*body, &ul)
	if err != nil {
		response.ErrorCode = com.WERR_INVALID_PARAM_JSON
		return
	}
	s.getBalanceList(response, ul)
}

func (s *WalletServerOperator) getBalanceList(response *com.BalanceListResponse, ul []com.UserId) {
	//ctx := req.Context()
	for i := 0; i < len(ul); i++ {
		uid := ul[i]
		s.userMapMutex.Lock()
		wallet, ok := s.userMap[uid]
		s.userMapMutex.Unlock()
		if !ok {
			continue
		}
		response.BalanceInfos = append(response.BalanceInfos, &com.BalanceInfo{UserId: uid, Amount: wallet.Balance})
	}
}

func (s *WalletServerOperator) subtractBalanceListTcp(vs *com.VSocket, requestId uint64, body *[]byte) {
	response := (&com.BalanceListResponse{}).Init()
	defer func() {
		res, _ := json.Marshal(response)
		vs.Response(requestId, res)
	}()

	param := com.UpdateBalanceListParam{}
	err := json.Unmarshal(*body, &param)
	if err != nil {
		response.ErrorCode = com.WERR_INVALID_PARAM_JSON
		return
	}
	s.subtractBalanceList(response, param.Infos)
}

func (s *WalletServerOperator) subtractBalanceList(response *com.BalanceListResponse, Infos []*com.AmountInfo) {

	for _, amountInfo := range Infos {
		s.userMapMutex.Lock()
		userWallet, userExist := s.userMap[amountInfo.UserId]
		s.userMapMutex.Unlock()
		if !userExist {
			msg := fmt.Sprintf("user id not exist %d", amountInfo.UserId)
			com.VUtils.PrintError(errors.New(msg))
			continue
		}
		if userWallet.Balance < com.Amount(amountInfo.Amount) {
			response.BalanceInfos = append(response.BalanceInfos, &com.BalanceInfo{UserId: userWallet.UserId, Amount: userWallet.Balance})
			continue
		}
		userWallet.BalanceMutex.Lock()
		userWallet.Balance -= com.Amount(amountInfo.Amount)
		userWallet.Hash = com.VUtils.HashAmount(uint64(userWallet.UserId), userWallet.Balance, userWallet.Currency)
		userWallet.NeedWriteDB = true
		userWallet.BalanceMutex.Unlock()

		response.BalanceInfos = append(response.BalanceInfos, &com.BalanceInfo{UserId: userWallet.UserId, Amount: userWallet.Balance})
	}
}

func (s *WalletServerOperator) addBalanceListTcp(vs *com.VSocket, requestId uint64, body *[]byte) {

	response := (&com.BalanceListResponse{}).Init()
	defer func() {
		res, err := json.Marshal(response)
		if err != nil {
			panic("addBalanceListTcp response fail")
		}
		//fmt.Println("addBalanceListTcp requestId ", requestId)
		//fmt.Println("addBalanceListTcp response ", string(res))
		vs.Response(requestId, res)
	}()

	param := com.UpdateBalanceListParam{}
	err := json.Unmarshal(*body, &param)
	if err != nil {
		response.ErrorCode = com.WERR_INVALID_PARAM_JSON
		return
	}
	s.addBalanceList(response, param.Infos)
}

func (s *WalletServerOperator) addBalanceList(response *com.BalanceListResponse, Infos []*com.AmountInfo) {

	for _, amountInfo := range Infos {
		s.userMapMutex.Lock()
		userWallet, userExist := s.userMap[amountInfo.UserId]
		s.userMapMutex.Unlock()
		if !userExist {
			msg := fmt.Sprintf("user id not exist %d", amountInfo.UserId)
			com.VUtils.PrintError(errors.New(msg))
			continue
		}

		userWallet.BalanceMutex.Lock()
		userWallet.Balance += com.Amount(amountInfo.Amount)
		userWallet.Hash = com.VUtils.HashAmount(uint64(userWallet.UserId), userWallet.Balance, userWallet.Currency)
		userWallet.NeedWriteDB = true
		userWallet.BalanceMutex.Unlock()

		response.BalanceInfos = append(response.BalanceInfos, &com.BalanceInfo{UserId: userWallet.UserId, Amount: userWallet.Balance})
	}

}

// ---------------------------

// API for single user --
func (s *WalletServerOperator) getBalanceTcp(vs *com.VSocket, requestId uint64, body *[]byte) {
	fmt.Println("wallet: getBalanceTcp requestId ", requestId)
	response := (&com.BalanceResponse{}).Init()
	defer func() {
		res, _ := json.Marshal(response)
		vs.Response(requestId, res)
	}()
	if len(*body) != 8 {
		response.ErrorCode = com.WERR_INVALID_PARAM_FORMAT
		return
	}
	uid := com.UserId(com.VUtils.BytesToUint64(body))
	fmt.Println("wallet: getBalanceTcp uid ", uid)
	s.getBalance(response, uid)
}

func (s *WalletServerOperator) getBalance(response *com.BalanceResponse, uid com.UserId) {
	//ctx := req.Context()
	s.userMapMutex.Lock()
	wallet, ok := s.userMap[uid]
	s.userMapMutex.Unlock()
	if !ok {
		msg := fmt.Sprintf("user id not exist %d", uid)
		com.VUtils.PrintError(errors.New(msg))
		response.ErrorCode = com.WERR_USER_NOT_EXIST
		return
	}
	response.BalanceInfo = &com.BalanceInfo{UserId: uid, Amount: wallet.Balance, Currency: wallet.Currency}
}

func (s *WalletServerOperator) widthdrawTcp(vs *com.VSocket, requestId uint64, body *[]byte) {
	response := (&com.BalanceResponse{}).Init()
	defer (func() {
		res, _ := json.Marshal(response)
		vs.Response(requestId, res)
	})()

	param := com.UpdateBalanceParam{}
	err := json.Unmarshal(*body, &param)
	if err != nil {
		response.ErrorCode = com.WERR_INVALID_PARAM_JSON
		return
	}
	s.widthdraw(response, param.Info)
}

func (s *WalletServerOperator) widthdraw(response *com.BalanceResponse, amountInfo *com.AmountInfo) {
	s.userMapMutex.Lock()
	userWallet, userExist := s.userMap[amountInfo.UserId]
	s.userMapMutex.Unlock()
	if !userExist {
		msg := fmt.Sprintf("user id not exist %d", amountInfo.UserId)
		com.VUtils.PrintError(errors.New(msg))
		response.ErrorCode = com.WERR_USER_NOT_EXIST
		return
	}
	if userWallet.Balance < com.Amount(amountInfo.Amount) {
		response.ErrorCode = com.WERR_INSUFFICIENT_BALANCE
		return
	}
	userWallet.BalanceMutex.Lock()
	userWallet.Balance -= com.Amount(amountInfo.Amount)
	userWallet.Hash = com.VUtils.HashAmount(uint64(userWallet.UserId), userWallet.Balance, userWallet.Currency)
	userWallet.NeedWriteDB = true
	userWallet.BalanceMutex.Unlock()

	response.BalanceInfo = &com.BalanceInfo{UserId: userWallet.UserId, Amount: userWallet.Balance}
	success := s.writeBetting(userWallet.UserId)
	if !success {
		response.ErrorCode = com.WERR_WRITE_BETTING_ERROR
	}
}

func (s *WalletServerOperator) maintenance() {
	if s.isMaintenance {
		return
	}
	s.isMaintenance = true
	Wallet.maintenance()
}

func (s *WalletServerOperator) depositTcp(vs *com.VSocket, requestId uint64, body *[]byte) {

	response := (&com.BalanceResponse{}).Init()
	defer func() {
		res, err := json.Marshal(response)
		if err != nil {
			panic("depositTcp response fail")
		}
		vs.Response(requestId, res)
	}()

	param := com.UpdateBalanceParam{}
	err := json.Unmarshal(*body, &param)
	if err != nil {
		response.ErrorCode = com.WERR_INVALID_PARAM_JSON
		return
	}
	s.deposit(response, param.Info)
}

func (s *WalletServerOperator) deposit(response *com.BalanceResponse, amountInfo *com.AmountInfo) {
	s.userMapMutex.Lock()
	userWallet, userExist := s.userMap[amountInfo.UserId]
	s.userMapMutex.Unlock()
	if !userExist {
		msg := fmt.Sprintf("user id not exist %d", amountInfo.UserId)
		com.VUtils.PrintError(errors.New(msg))
		response.ErrorCode = com.WERR_USER_NOT_EXIST
		return
	}

	userWallet.BalanceMutex.Lock()
	userWallet.Balance += com.Amount(amountInfo.Amount)
	userWallet.Hash = com.VUtils.HashAmount(uint64(userWallet.UserId), userWallet.Balance, userWallet.Currency)
	userWallet.NeedWriteDB = true
	userWallet.BalanceMutex.Unlock()

	response.BalanceInfo = &com.BalanceInfo{UserId: userWallet.UserId, Amount: userWallet.Balance}
	success := s.writeBetting(userWallet.UserId)
	if !success {
		response.ErrorCode = com.WERR_WRITE_BETTING_ERROR
	}
}

func (s *WalletServerOperator) createOrGetBettingRecord(param *com.BettingParam) *com.BettingRecord {

	dbBettingId := param.BettingId
	if dbBettingId > 0 {
		s.bettingMapMutex.Lock()
		bettingRecord, has := s.bettingMap[dbBettingId]
		s.bettingMapMutex.Unlock()
		if has {
			return bettingRecord
		} else {
			betting := com.BettingRecord{}
			query := "SELECT id, gameid, gamenumber, roomid, userid, roundid, betdetail, result, payout, payedout, h FROM betting WHERE id=?"
			hash := ""
			row := s.db.QueryRow(query, dbBettingId)
			err := row.Scan(&betting.BettingId, &betting.GameId, &betting.GameNumber, &betting.RoomId, &betting.UserId, &betting.RoundId, &betting.BetDetail, &betting.Result, &betting.Payout, &betting.Payedout, &hash)
			if err != nil {
				return nil
			}
			str := fmt.Sprintf("%d_%s_%s_%d_%d_%s_%s_%f_%d", betting.GameNumber, betting.GameId, betting.RoomId, betting.RoundId, betting.UserId, betting.BetDetail, betting.Result, betting.Payout, betting.Payedout)
			if hash != com.VUtils.HashString(str) {
				fmt.Println("wrong hash for com.GameNumber: ", betting.GameNumber, hash)
				s.Stop()
				return nil
			}
			s.bettingMapMutex.Lock()
			s.bettingMap[dbBettingId] = &betting
			s.bettingMapMutex.Unlock()
			return &betting
		}
	}

	// find existing dbBettingId
	row := s.db.QueryRow("SELECT id FROM betting WHERE gameid=? AND roomid=? AND gamenumber=? AND userid=?", param.GameId, param.RoomId, param.GameNumber, param.UserId)
	err := row.Scan(&dbBettingId)
	if err != nil {
		//fmt.Println("createOrGetcom.BettingRecord not found dbBettingId ", err)
		dbBettingId = 0
	} else {
		//fmt.Println("createOrGetcom.BettingRecord found dbBettingId === ", dbBettingId)
	}

	bettingRecord := com.BettingRecord{}
	bettingRecord.UserId = param.UserId
	bettingRecord.BettingId = dbBettingId
	// create a new record in table betting
	if bettingRecord.BettingId == 0 {
		tx, err := s.db.Begin()
		if err != nil {
			com.VUtils.PrintError(err)
			return nil
		}
		str := fmt.Sprintf("%d_%s_%s_%d_%d_%s_%s_%f_%d", param.GameNumber, param.GameId, param.RoomId, param.RoundId, param.UserId, "", "", bettingRecord.Payout, bettingRecord.Payedout)
		hash := com.VUtils.HashString(str)
		query := "INSERT INTO betting (gamenumber, gameid , roomid, userid, roundid, betdetail, result, payout, payedout, h) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		response, err2 := tx.Exec(query, param.GameNumber, param.GameId, param.RoomId, param.UserId, param.RoundId, "", "", bettingRecord.Payout, bettingRecord.Payedout, hash)
		if err2 != nil {
			tx.Rollback()
			com.VUtils.PrintError(err2)
			return nil
		}
		createdBettingId, err3 := response.LastInsertId()
		if err3 != nil {
			tx.Rollback()
			com.VUtils.PrintError(err3)
			return nil
		}
		err4 := tx.Commit()
		if err4 != nil {
			com.VUtils.PrintError(err4)
			s.maintenance()
			return nil
		}
		bettingRecord.BettingId = com.BettingId(createdBettingId)
		// each user only can create 1 betting record  each gameNumber. MUST CHECK
		//fmt.Println("each user only can create 1 betting record  each gameNumber. MUST CHECK")
		//fmt.Println("create new bettingRecord.BettingId ==== ", bettingRecord.BettingId, " param.GameNumber ", param.GameNumber)
	}
	return &bettingRecord
}

func (s *WalletServerOperator) saveBettingTcp(vs *com.VSocket, requestId uint64, body *[]byte) {

	response := &com.BettingResponse{}
	defer func() {
		res, err := json.Marshal(response)
		if err != nil {
			panic("saveBettingTcp response fail")
		}
		vs.Response(requestId, res)
	}()

	param := com.BettingParam{}
	err := json.Unmarshal(*body, &param)
	if err != nil {
		response.ErrorCode = com.WERR_INVALID_PARAM_JSON
		fmt.Println("saveBettingTcp --- 1")
		return
	}
	s.userMapMutex.Lock()
	userWallet, userExist := s.userMap[param.UserId]
	s.userMapMutex.Unlock()
	if !userExist {
		fmt.Println("saveBettingTcp --- 2")
		msg := fmt.Sprintf("user id not exist %d", param.UserId)
		com.VUtils.PrintError(errors.New(msg))
		response.ErrorCode = com.WERR_USER_NOT_EXIST
		return
	}

	bettingRecord := s.createOrGetBettingRecord(&param)
	if bettingRecord == nil {
		fmt.Println("saveBettingTcp --- 3")
		response.ErrorCode = com.WERR_CREATE_BETTING_ERROR
		return
	}

	// check change balance valid ---
	// no need to lock on bettingRecord, bettingRecord already be sequence update by game operators
	userWallet.BalanceMutex.Lock()
	isBalanceValid := userWallet.Balance+param.BalanceChange > 0
	if isBalanceValid {
		// copy data from param
		bettingRecord.Copy(&param)
		// update user balance

		userWallet.Balance += param.BalanceChange
		userWallet.Hash = com.VUtils.HashAmount(uint64(userWallet.UserId), userWallet.Balance, userWallet.Currency)
		userWallet.NeedWriteDB = true

		response.ErrorCode = 0 // success

		s.bettingMapMutex.Lock()
		// override old com.BettingId field
		s.bettingMap[bettingRecord.BettingId] = bettingRecord
		s.bettingMapMutex.Unlock()
	} else {
		response.ErrorCode = com.WERR_INSUFFICIENT_BALANCE
	}
	response.UserId = param.UserId
	response.Balance = userWallet.Balance
	response.BettingId = bettingRecord.BettingId
	userWallet.BalanceMutex.Unlock()

}

// ---------------------------

func (s *WalletServerOperator) writeBetting(userId com.UserId) bool {
	// TODO: check write DB work correctly for server performance, dont write the same value
	// use transaction
	tx, err := s.db.Begin()
	if err != nil {
		com.VUtils.PrintError(err)
		s.maintenance()
		return false
	}
	// write bettings --
	userBettings := []*com.BettingRecord{}
	s.bettingMapMutex.Lock()
	for _, betting := range s.bettingMap {
		betting.NeedWriteDBMutex.Lock()
		if betting.UserId == userId && betting.NeedWriteDB {
			userBettings = append(userBettings, betting)
			betting.NeedWriteDB = false
		}
		betting.NeedWriteDBMutex.Unlock()
	}
	s.bettingMapMutex.Unlock()
	for _, betting := range userBettings {
		str := fmt.Sprintf("%d_%s_%s_%d_%d_%s_%s_%f_%d", betting.GameNumber, betting.GameId, betting.RoomId, betting.RoundId, betting.UserId, betting.BetDetail, betting.Result, betting.Payout, betting.Payedout)
		hash := com.VUtils.HashString(str)

		_, err := tx.Exec("UPDATE betting SET betdetail=?, result=?, payout=?, payedout=?, h=? WHERE id=?", betting.BetDetail, betting.Result, betting.Payout, betting.Payedout, hash, betting.BettingId)
		if err != nil {
			fmt.Println()
			tx.Rollback()
			com.VUtils.PrintError(err)
			s.maintenance()
			return false
		}

	}
	// write balance --
	s.userMapMutex.Lock()
	walletInfo, userExist := s.userMap[userId]
	s.userMapMutex.Unlock()
	if !userExist {
		msg := fmt.Sprintf("scheduleWriteBetting user id not exist %d", userId)
		com.VUtils.PrintError(errors.New(msg))
		return false
	}
	if walletInfo.NeedWriteDB {
		// for game performance, dont wait for mysql query ---
		walletInfo.BalanceMutex.Lock()
		walletInfo.NeedWriteDB = false
		walletInfo.BalanceMutex.Unlock()
		// ---
		_, err := tx.Exec("UPDATE wallet SET balance=?, h=? WHERE userid=?", walletInfo.Balance, walletInfo.Hash, userId)
		if err != nil {
			tx.Rollback()
			com.VUtils.PrintError(err)
			return false
		}
	}
	err4 := tx.Commit()
	if err4 != nil {
		com.VUtils.PrintError(err4)
		s.maintenance()
		return false
	}
	return true
}

func (s *WalletServerOperator) clearBettingTcp(vs *com.VSocket, requestId uint64, body *[]byte) {
	response := (&com.BaseWalletResponse{}).Init()
	defer func() {
		res, err := json.Marshal(response)
		if err != nil {
			panic("clearBettingTcp response fail")
		}
		vs.Response(requestId, res)
	}()

	gameNumber := com.VUtils.BytesToUint64(body)
	response.IntVal = s.clearBetting(com.GameNumber(gameNumber))
}

func (s *WalletServerOperator) clearBetting(gameNumber com.GameNumber) int {
	deleteIds := []com.BettingId{}
	s.bettingMapMutex.Lock()
	for _, betting := range s.bettingMap {
		if betting.GameNumber == gameNumber {
			deleteIds = append(deleteIds, betting.BettingId)
		}
	}

	for _, bettingId := range deleteIds {
		delete(s.bettingMap, bettingId)
	}
	s.bettingMapMutex.Unlock()
	return len(deleteIds)
}

func (s *WalletServerOperator) queryBettingTcp(vs *com.VSocket, requestId uint64, body *[]byte) {
	response := &com.QueryBettingResponse{}
	defer func() {
		res, err := json.Marshal(response)
		if err != nil {
			panic("queryBettingTcp response fail")
		}
		vs.Response(requestId, res)
	}()

	gameNumber := com.VUtils.BytesToUint64(body)
	response.Bettings = s.queryBetting(com.GameNumber(gameNumber))
}

func (s *WalletServerOperator) queryBetting(gameNumber com.GameNumber) []*com.BettingRecord {
	bettings := []*com.BettingRecord{}
	query := "SELECT id, gameid, gamenumber, roomid, userid, roundid, betdetail, result, payout, payedout, h FROM betting WHERE gamenumber=?"
	rows, err := s.db.Query(query, gameNumber)
	if err != nil {
		com.VUtils.PrintError(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		betting := com.BettingRecord{}
		hash := ""
		err := rows.Scan(&betting.BettingId, &betting.GameId, &betting.GameNumber, &betting.RoomId, &betting.UserId, &betting.RoundId, &betting.BetDetail, &betting.Result, &betting.Payout, &betting.Payedout, &hash)

		if err != nil {
			com.VUtils.PrintError(err)
			s.maintenance()
			return nil
		}
		str := fmt.Sprintf("%d_%s_%s_%d_%d_%s_%s_%f_%d", betting.GameNumber, betting.GameId, betting.RoomId, betting.RoundId, betting.UserId, betting.BetDetail, betting.Result, betting.Payout, betting.Payedout)

		if hash != com.VUtils.HashString(str) {
			fmt.Println("wrong hash for com.GameNumber ", betting.GameNumber, hash)
			s.Stop()
			return nil
		}
		bettings = append(bettings, &betting)
	}
	return bettings
}

func (s *WalletServerOperator) scheduleWriteDB(dt float64) {
	for userId := range s.userMap {
		s.writeBetting(userId)
	}
}

func (s *WalletServerOperator) Stop() {
	s.scheduleWriteDB(0)
	if s.db != nil {
		s.db.Close()
	}
}

func (s *WalletServerOperator) loadHistoryTcp(vs *com.VSocket, requestId uint64, body *[]byte) {
	response := &com.HistoryResponse{}
	defer func() {
		res, err := json.Marshal(response)
		if err != nil {
			panic("loadHistoryTcp response fail")
		}
		vs.Response(requestId, res)
	}()

	param := com.WalletHistoryParam{}
	err := json.Unmarshal(*body, &param)
	if err != nil {
		com.VUtils.PrintError(err)
		return
	}
	response.Items = s.loadHistory(param.GameId, param.UserId, param.PageInd)
}

func (s *WalletServerOperator) loadHistory(gameId com.GameId, userId com.UserId, page uint32) []*com.HistoryRecord {
	startRow := page * uint32(com.HIS_PAGE_SIZE)
	endRow := (page + 1) * uint32(com.HIS_PAGE_SIZE)
	query := "SELECT gamenumber, roundid, betdetail, result, payout, updatetime FROM betting WHERE gameid=? AND userid=? AND payedout=1 ORDER BY updatetime DESC LIMIT ?, ?"
	rows, err := s.db.Query(query, gameId, userId, startRow, endRow)
	if err != nil {
		com.VUtils.PrintError(err)
		return nil
	}
	defer rows.Close()
	items := []*com.HistoryRecord{}
	for rows.Next() {
		bet := com.HistoryRecord{}
		time := time.Time{}
		err := rows.Scan(&bet.GameNumber, &bet.RoundId, &bet.BetDetail, &bet.Result, &bet.Payout, &time)
		if err != nil {
			com.VUtils.PrintError(err)
			s.maintenance()
			return nil
		}
		//fmt.Println(time)
		bet.Time = time.UnixMilli()
		items = append(items, &bet)
	}
	return items
}
