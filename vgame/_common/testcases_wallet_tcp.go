package com

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type TestWalletTcp struct {
	operatorId  OperatorID
	Token       string
	vs          *VSocket
	encryptType EncryptType
}

func (t *TestWalletTcp) init() *TestWalletTcp {
	t.operatorId = "001"
	t.vs = nil
	t.encryptType = GAME_ENCRYPT
	return t
}

func (t *TestWalletTcp) createAcc() {

	vs := t.vs
	if vs == nil {
		conn, err := net.Dial("tcp", WALLET_HOST+":"+WALLET_PORT)
		if err != nil {
			panic(err)
		}

		vs = (&VSocket{}).Init(conn, func(vs *VSocket, requestId uint64, data []byte) {
		}, nil, true, 1, t.encryptType)
		serverType := uint8(1)
		bytes := []byte{serverType}
		cmd := uint16(WCMD_REGISTER_CONN)
		pres := VUtils.Uint16ToBytes(cmd)
		bytes = append(pres, bytes...)

		vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
			bytes := VUtils.WalletLocalMessageString(t.operatorId, WCMD_CREATE_ACC, "USDC")
			vs.Send(bytes, nil, nil)
		}, nil)
	}

}

func (c *TestWalletTcp) createAccMulti(num int) {
	for i := 0; i < num; i++ {
		c.createAcc()
	}
}

func (t *TestWalletTcp) getBalances(vs *VSocket, ul []UserId, wg *sync.WaitGroup) {
	//fmt.Println("getBalances ===")

	bytes := VUtils.Uint16ToBytes(WCMD_GET_BALANCE_LIST)
	body, _ := json.Marshal(ul)
	bytes = append(bytes, body...)

	ops := []byte(t.operatorId)
	bytes = append(ops, bytes...)

	vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
		response := BalanceListResponse{}
		err := json.Unmarshal(resData, &response)
		if err != nil {
			panic("response getBalances fail")
		}
		if len(response.BalanceInfos) > 0 {
			//fmt.Println("getBalances response: ", response.BalanceInfos[0].UserId, response.BalanceInfos[0].Amount)
		}
		wg.Done()
	}, nil)
}

func (t *TestWalletTcp) addBalances(vs *VSocket, amtInfors []*AmountInfo, wg *sync.WaitGroup) {

	bytes := VUtils.Uint16ToBytes(WCMD_ADD_BALANCE_LIST)
	param := UpdateBalanceListParam{}
	param.Infos = amtInfors
	body, _ := json.Marshal(param)
	bytes = append(bytes, body...)

	ops := []byte(t.operatorId)
	bytes = append(ops, bytes...)

	vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
		response := BalanceListResponse{}
		err := json.Unmarshal(resData, &response)
		if err != nil {
			panic("response addBalances fail")
		}
		wg.Done()
	}, nil)
}

func (t *TestWalletTcp) subtractBalances(vs *VSocket, amtInfors []*AmountInfo, wg *sync.WaitGroup) {

	bytes := VUtils.Uint16ToBytes(WCMD_SUBTRACT_BALANCE_LIST)
	param := UpdateBalanceListParam{}
	param.Infos = amtInfors
	body, _ := json.Marshal(param)
	bytes = append(bytes, body...)

	ops := []byte(t.operatorId)
	bytes = append(ops, bytes...)

	vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
		response := BalanceListResponse{}
		err := json.Unmarshal(resData, &response)
		if err != nil {
			panic("response subtractBalances fail")
		}
		wg.Done()
	}, nil)
}

func (c *TestWalletTcp) getBalanceMulti(threadNum int, loopTime int) {
	g := sync.WaitGroup{}
	g.Add(threadNum * loopTime)
	for i := 0; i < threadNum; i++ {
		go func() {
			for j := 0; j < loopTime; j++ {
				vs := c.vs
				if vs == nil {
					conn, err := net.Dial("tcp", WALLET_HOST+":"+WALLET_PORT)
					if err != nil {
						panic(err)
					}

					vs = (&VSocket{}).Init(conn, func(vs *VSocket, requestId uint64, data []byte) {
					}, nil, true, 1, c.encryptType)
					serverType := uint8(1)
					bytes := []byte{serverType}
					cmd := uint16(WCMD_REGISTER_CONN)
					pres := VUtils.Uint16ToBytes(cmd)
					bytes = append(pres, bytes...)

					vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
						c.getBalances(vs, []UserId{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, &g)
					}, nil)
				} else {
					c.getBalances(vs, []UserId{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, &g)
				}
			}
		}()
	}

	g.Wait()
}

func (c *TestWalletTcp) addBalanceMulti(threadNum int, loopTime int, accNum int, amount Amount) {
	g := sync.WaitGroup{}
	g.Add(threadNum * loopTime)

	amtInfors := []*AmountInfo{}
	for i := 1; i <= accNum; i++ {
		amtInfors = append(amtInfors, &AmountInfo{UserId: UserId(i), Amount: amount})
	}

	for i := 0; i < threadNum; i++ {
		//time.Sleep(time.Second * 1.0)
		//fmt.Println("start thread ", i)
		go func() {
			for j := 0; j < loopTime; j++ {
				vs := c.vs
				if vs == nil {

					conn, err := net.Dial("tcp", WALLET_HOST+":"+WALLET_PORT)
					if err != nil {
						panic(err)
					}

					vs = (&VSocket{}).Init(conn, func(vs *VSocket, requestId uint64, data []byte) {
					}, nil, true, 1, c.encryptType)
					serverType := uint8(1)
					bytes := []byte{serverType}
					cmd := uint16(WCMD_REGISTER_CONN)
					pres := VUtils.Uint16ToBytes(cmd)
					bytes = append(pres, bytes...)

					vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
						c.addBalances(vs, amtInfors, &g)
					}, nil)
				} else {
					c.addBalances(vs, amtInfors, &g)
				}
			}
		}()
	}

	g.Wait()
}

func (c *TestWalletTcp) subtractBalanceMulti(threadNum int, loopTime int, accNum int, amount Amount) {
	g := sync.WaitGroup{}
	g.Add(threadNum * loopTime)

	amtInfors := []*AmountInfo{}
	for i := 1; i <= accNum; i++ {
		amtInfors = append(amtInfors, &AmountInfo{UserId: UserId(i), Amount: amount})
	}

	for i := 0; i < threadNum; i++ {
		go func() {
			for j := 0; j < loopTime; j++ {
				vs := c.vs
				if vs == nil {
					conn, err := net.Dial("tcp", WALLET_HOST+":"+WALLET_PORT)
					if err != nil {
						panic(err)
					}

					vs = (&VSocket{}).Init(conn, func(vs *VSocket, requestId uint64, data []byte) {
					}, nil, true, 1, c.encryptType)
					serverType := uint8(1)
					bytes := []byte{serverType}
					cmd := uint16(WCMD_REGISTER_CONN)
					pres := VUtils.Uint16ToBytes(cmd)
					bytes = append(pres, bytes...)

					vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
						c.subtractBalances(vs, amtInfors, &g)
					}, nil)
				} else {
					c.subtractBalances(vs, amtInfors, &g)
				}

			}
		}()
	}

	g.Wait()
}

func loadTest() {
	// ----
	for i := 0; i < 10; i++ {
		test := i % 3
		switch test {
		case 0:
			startTime := time.Now().UnixMilli()
			TestWalletTcpClient.getBalanceMulti(10, 10)
			endTime := time.Now().UnixMilli()
			fmt.Println("get balance process in ms ", (endTime - startTime))
		case 1:
			startTime := time.Now().UnixMilli()
			TestWalletTcpClient.addBalanceMulti(10, 10, 10, 2)
			endTime := time.Now().UnixMilli()
			fmt.Println("add balance process in ms ", (endTime - startTime))
		case 2:
			startTime := time.Now().UnixMilli()
			TestWalletTcpClient.subtractBalanceMulti(10, 10, 10, 1)
			endTime := time.Now().UnixMilli()
			fmt.Println("subtract balance process in ms ", (endTime - startTime))
		}

		time.Sleep(1 * time.Second)
	}

	time.Sleep(1 * time.Second)
	fmt.Println("test done")
}

/*
// delete alls
SET SQL_SAFE_UPDATES = 0;
DELETE FROM wallet;
SET SQL_SAFE_UPDATES = 1;
*/

func (t *TestWalletTcp) CreateTestAcc() {
	t.createAccMulti(2000)
}

func (t *TestWalletTcp) AddBalance() {
	TestWalletTcpClient.addBalanceMulti(1, 1, 1000, 20000)
}

func (t *TestWalletTcp) DoTest() {
	if 1 > 0 {
		fmt.Println("test success")
		return
	}

	g := sync.WaitGroup{}
	g.Add(1)
	//TestWalletTcpClient.getBalances([]UserId{1, 2}, &g)

	// NOTE: the test create a new tcp connection so it will take time for connect also
	conn, err := net.Dial("tcp", WALLET_HOST+":"+WALLET_PORT)
	if err != nil {
		panic(err)
	}

	useSingleSocket := false
	if useSingleSocket {
		t.vs = (&VSocket{}).Init(conn, func(vs *VSocket, requestId uint64, data []byte) {
		}, nil, true, 2, t.encryptType)
		serverType := uint8(1)
		bytes := []byte{serverType}
		cmd := uint16(WCMD_REGISTER_CONN)
		pres := VUtils.Uint16ToBytes(cmd)
		bytes = append(pres, bytes...)
		t.vs.Send(bytes, func(vs *VSocket, requestId uint64, resData []byte) {
			fmt.Println("got res ", string(resData))
			loadTest()
		}, nil)
	} else {
		loadTest()
	}

}

var TestWalletTcpClient = (&TestWalletTcp{}).init()
