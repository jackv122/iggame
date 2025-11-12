package com

import "sync"

type QueryBettingResponse struct {
	ErrorCode int
	ErrorMsg  string
	Bettings  []*BettingRecord
}

type BaseWalletResponse struct {
	ErrorCode int
	ErrorMsg  string
	IntVal    int
}

func (res *BaseWalletResponse) Init() *BaseWalletResponse {
	res.ErrorCode = 0
	res.ErrorMsg = ""
	res.IntVal = 0
	return res
}

type HistoryRecord struct {
	GameNumber GameNumber
	RoundId    RoundId
	BetDetail  string
	Result     string
	Payout     Amount
	Time       int64
}

type HistoryResponse struct {
	ErrorCode   int
	ErrorMsg    string
	Items       []*HistoryRecord
	GameDetails []*GameDetailItem
}

func (res *HistoryResponse) Init() *HistoryResponse {
	res.ErrorCode = 0
	res.ErrorMsg = ""
	res.Items = []*HistoryRecord{}
	res.GameDetails = []*GameDetailItem{}
	return res
}

type BalanceResponse struct {
	ErrorCode   int
	ErrorMsg    string
	BalanceInfo *BalanceInfo
}

func (res *BalanceResponse) Init() *BalanceResponse {
	res.ErrorCode = 0
	res.ErrorMsg = ""
	res.BalanceInfo = &BalanceInfo{}
	return res
}

type BettingResponse struct {
	ErrorCode int
	ErrorMsg  string
	BettingId BettingId
	UserId    UserId
	Balance   Amount
}

type BalanceListResponse struct {
	ErrorCode    int
	ErrorMsg     string
	BalanceInfos []*BalanceInfo
}

func (res *BalanceListResponse) Init() *BalanceListResponse {
	res.ErrorCode = 0
	res.ErrorMsg = ""
	res.BalanceInfos = []*BalanceInfo{}
	return res
}

type UpdateBalanceParam struct {
	Token    string
	Info     *AmountInfo
	Checksum uint64
}

func (param *UpdateBalanceParam) Init() *UpdateBalanceParam {
	param.Info = &AmountInfo{}
	return param
}

type UpdateBalanceListParam struct {
	Token    string
	Infos    []*AmountInfo
	Checksum uint64
}

func (param *UpdateBalanceListParam) Init() *UpdateBalanceListParam {
	param.Infos = []*AmountInfo{}
	return param
}

type GetChecksumParam struct {
}

type GetChecksumResponse struct {
	Token    string
	Checksum uint64
}

type CreateAccResponse struct {
	ErrorCode int
	ErrorMsg  string
	UserId    UserId
}

func (res *CreateAccResponse) Init() *CreateAccResponse {
	res.ErrorCode = 0
	res.ErrorMsg = ""
	return res
}

type WalletHistoryParam struct {
	GameId  GameId
	UserId  UserId
	PageInd uint32
}

type BettingParam struct {
	// params ------------------------
	BettingId     BettingId
	GameId        GameId
	GameNumber    GameNumber
	RoundId       RoundId
	RoomId        RoomId
	UserId        UserId
	BetDetail     string
	Result        string
	Payout        Amount
	Payedout      uint8  // 0 or 1
	BalanceChange Amount // it can be the change amount when (bet/undo bet/payout)
	// -------------------------------
}

type BettingRecord struct {
	// params ------------------------
	BettingId     BettingId
	GameId        GameId
	GameNumber    GameNumber
	RoundId       RoundId
	RoomId        RoomId
	UserId        UserId
	BetDetail     string
	Result        string
	Payout        Amount
	Payedout      uint8  // 0 or 1
	BalanceChange Amount // it can be the change amount when (bet/undo bet/payout)
	// -------------------------------

	// private data
	NeedWriteDB      bool
	NeedWriteDBMutex sync.Mutex
}

func (r *BettingRecord) Copy(param *BettingParam) {
	r.GameId = param.GameId
	r.GameNumber = param.GameNumber
	r.RoundId = param.RoundId
	r.RoomId = param.RoomId
	r.UserId = param.UserId
	r.BetDetail = param.BetDetail
	r.Result = param.Result
	r.Payout = param.Payout
	r.Payedout = param.Payedout
	r.BalanceChange = param.BalanceChange

	r.NeedWriteDBMutex.Lock()
	r.NeedWriteDB = true
	r.NeedWriteDBMutex.Unlock()
}

type UserWallet struct {
	UserId       UserId
	Balance      Amount
	Hash         string
	Currency     Currency
	BalanceMutex sync.Mutex
	NeedWriteDB  bool
}

func (u *UserWallet) Init() *UserWallet {
	return u
}
