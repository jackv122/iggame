package com

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"net"
	"os"
)

var (
	vtestEncryptPort = PROXY_TCP_PORT
)

type TestVSocketObj struct {
	Cmd int
	Msg string
}

type TestVSocketT struct {
	host          string
	port          string
	callbackCount int
	callbackSent  int
	mutex         sync.Mutex
	encryptType   EncryptType
}

var TestVSocket = (&TestVSocketT{}).init()

func (t *TestVSocketT) init() *TestVSocketT {
	t.host = "localhost"
	t.port = vtestEncryptPort
	t.encryptType = 1
	return t
}

func (t *TestVSocketT) testLoopParseMsg_1() {

	// msg 1
	msg := []byte{1, 2, 3}

	vs := (&VSocket{}).Init(nil, func(vs *VSocket, requestId uint64, data []byte) {
		if requestId == 1 && string(data) == string(msg) {
			fmt.Println("testLoopParseMsg_1 : PASSED", "received: ", data)
		} else {
			fmt.Println("testLoopParseMsg_1 : fail")
		}
	}, nil, false, 1, t.encryptType)
	bytes := msg
	bt := VUtils.Uint64ToBytes(vs.checksum)
	requestId := uint64(1)
	isResponse := []byte{0}
	bt = append(bt, isResponse...)
	bt = append(bt, VUtils.Uint64ToBytes(requestId)...)
	bytes = append(bt, bytes...)

	Encrypt(bytes, PROXY_ENCRYPT)
	l := uint16(len(bytes))
	bytes = append(VUtils.Uint16ToBytes(l), bytes...)

	// more redundant datas
	bytes = append(bytes, VUtils.Uint16ToBytes(2)...)

	vs.msgBytes = bytes
	vs.loopParseMsg()
}

func (t *TestVSocketT) testLoopParseMsg_2() {

	// msg1 1
	msg1 := []byte{1, 2, 3}
	msg2 := []byte{4, 5, 6, 7}

	vs := (&VSocket{}).Init(nil, func(vs *VSocket, requestId uint64, data []byte) {
		if requestId == 1 && string(data) == string(msg1) {
			fmt.Println("testLoopParseMsg_2 : PASSED  ", "received: ", data)
		} else if requestId == 2 && string(data) == string(msg2) {
			fmt.Println("testLoopParseMsg_2 : PASSED  ", "received: ", data)
		} else {
			fmt.Println("testLoopParseMsg_2 : fail")
		}
	}, nil, true, 1, t.encryptType)

	bytes := []byte{}
	// write msg 1 --------

	bt := VUtils.Uint64ToBytes(vs.checksum)
	requestId := uint64(1)
	isResponse := []byte{0}
	bt = append(bt, isResponse...)
	bt = append(bt, VUtils.Uint64ToBytes(requestId)...)
	bt = append(bt, msg1...)

	Encrypt(bt, PROXY_ENCRYPT)
	l := uint16(len(bt))
	bt = append(VUtils.Uint16ToBytes(l), bt...)

	bytes = append(bytes, bt...)

	// write msg 2 --------

	bt = VUtils.Uint64ToBytes(vs.checksum)
	requestId = uint64(2)
	isResponse = []byte{0}
	bt = append(bt, isResponse...)
	bt = append(bt, VUtils.Uint64ToBytes(requestId)...)

	bt = append(bt, msg2...)

	Encrypt(bt, PROXY_ENCRYPT)
	l = uint16(len(bt))
	bt = append(VUtils.Uint16ToBytes(l), bt...)

	bytes = append(bytes, bt...)

	// more redundant datas ---
	bytes = append(bytes, VUtils.Uint16ToBytes(2)...)

	vs.msgBytes = bytes
	vs.loopParseMsg()
}

func (t *TestVSocketT) testSendWithCallback(vs *VSocket, cmd int, msg string) {
	if cmd > 0 {
		t.mutex.Lock()
		t.callbackSent++
		t.mutex.Unlock()
	}
	obj := TestVSocketObj{}
	// need a response
	obj.Cmd = cmd
	obj.Msg = msg
	sendData, _ := json.Marshal(obj)

	vs.Send(sendData, func(vs *VSocket, requestId uint64, resData []byte) {
		t.mutex.Lock()
		t.callbackCount++
		t.mutex.Unlock()
		fmt.Println("response from server: "+string(resData)+" callbackCount ", t.callbackCount)
		if t.callbackSent == t.callbackCount {
			fmt.Println("test vsocket checksum & callback: PASSED")
		}
	}, func() {
		fmt.Println("test vsocket timeout for msg: ", msg)
	})
}

func (t *TestVSocketT) onMsgHdl(vs *VSocket, requestId uint64, data []byte) {
	obj := TestVSocketObj{}
	json.Unmarshal(data, &obj)
	fmt.Println("server got msg: " + obj.Msg)
	if obj.Cmd == 0 {
	} else if obj.Cmd == 1 {
	} else if obj.Cmd == 2 {
		time.Sleep(7 * time.Second)
	} else if obj.Cmd == 3 {
		time.Sleep(1 * time.Second)
	} else {
		time.Sleep(time.Duration(VUtils.GetRandInt(5)) * time.Second)
	}
	if obj.Cmd != 0 {
		vs.Response(requestId, []byte("server response for: "+obj.Msg))
	}
}

func (t *TestVSocketT) onCloseHdl(vs *VSocket) {

}

func (t *TestVSocketT) processClient(conn net.Conn) {
	// it is a server so let's use a multithread VSocket
	(&VSocket{}).Init(conn, t.onMsgHdl, t.onCloseHdl, true, 10, t.encryptType)
}

func (t *TestVSocketT) startSocketServer() {
	fmt.Println("Server socket Running...")
	server, err := net.Listen("tcp", t.host+":"+t.port)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer server.Close()
	fmt.Println("Waiting for client...")
	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("client connected")
		go t.processClient(conn)
	}
}

func (t *TestVSocketT) DoTest() {
	go func() {
		//time.Sleep(3 * time.Second)
		//TestVSocket.testLoopParseMsg_1()
		//TestVSocket.testLoopParseMsg_2()
		t := TestVSocket
		fmt.Println("start test vsocket ", t.host, t.port)
		var conn, err = net.Dial("tcp", t.host+":"+t.port)
		if err != nil {
			panic(err)
		}
		vs := (&VSocket{}).Init(conn, func(vs *VSocket, requestId uint64, data []byte) {
		}, nil, true, 10, t.encryptType)

		TestVSocket.testSendWithCallback(vs, 0, "msg0")
		return
		TestVSocket.testSendWithCallback(vs, 1, "msg1")
		TestVSocket.testSendWithCallback(vs, 2, "msg2")
		TestVSocket.testSendWithCallback(vs, 3, "msg3")

		time.Sleep(7 * time.Second)
		fmt.Println("start test gorountine with sending ----")
		// test goroutine sending ---
		for i := 0; i < 10; i++ {
			go TestVSocket.testSendWithCallback(vs, 4, "msg1")
			go TestVSocket.testSendWithCallback(vs, 5, "msg2")
			go TestVSocket.testSendWithCallback(vs, 6, "msg3")
		}
		// expected 33 callback
	}()
	//TestVSocket.startSocketServer()
}
