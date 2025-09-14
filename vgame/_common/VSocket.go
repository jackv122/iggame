package com

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"unsafe"
)

const (
	VSOCKET_MSG_MAX_LENGTH = 100 * 1024 * 1024
)

type VSocketTask struct {
	bytes     *[]byte
	cb        *VSocketCallback
	requestId uint64
}

type VSocketCallback struct {
	succesHdl    func(vs *VSocket, requestId uint64, resBytes []byte)
	timeoutTimer *time.Timer
}

type VSocket struct {
	taskList         chan *VSocketTask
	onCloseHdl       func(vs *VSocket)
	onMsgHdl         func(vs *VSocket, requestId uint64, resBytes []byte)
	mutex            sync.Mutex
	responseMapMutex sync.Mutex
	checksum         uint64
	receiveChecksum  uint64
	// it is interface, dont user pointer to it
	conn           net.Conn
	responseHdlMap map[uint64]*VSocketCallback
	msgBytes       []byte
	contentSize    uint32
	closed         bool
	syncCk         bool
	isValid        bool
	EncryptType    EncryptType
	UserData       unsafe.Pointer
}

func (vs *VSocket) Init(conn net.Conn, onMsgHdl func(vs *VSocket, requestId uint64, resBytes []byte), closeHdl func(vs *VSocket), syncCk bool, callbackThreadNum uint8, encryptType EncryptType) *VSocket {
	vs.taskList = make(chan *VSocketTask)
	vs.onMsgHdl = onMsgHdl
	vs.onCloseHdl = closeHdl
	vs.checksum = 0
	vs.receiveChecksum = 0
	vs.contentSize = 0
	vs.conn = conn
	vs.syncCk = syncCk
	vs.msgBytes = []byte{}
	vs.isValid = true
	vs.EncryptType = encryptType
	vs.responseHdlMap = map[uint64]*VSocketCallback{}
	vs.UserData = nil
	if conn == nil {
		return vs
	}
	// init worker threads ----
	for i := uint8(0); i < callbackThreadNum; i++ {
		go func() {
			for task := range vs.taskList {
				if task.cb != nil {
					task.cb.succesHdl(vs, task.requestId, *task.bytes)
				} else {
					vs.onMsgHdl(vs, task.requestId, *task.bytes)
				}
			}
		}()
	}
	// create a goroutine for reading message until the conn is closed

	go func() {
		buffer := make([]byte, 1024)
		for {
			// got new buffer
			l, err := conn.Read(buffer)

			// black hole process, do nothing
			if !vs.isValid {
				continue
			}
			vs.msgBytes = append(vs.msgBytes, buffer[:l]...)

			vs.loopParseMsg()

			if l == 0 || err == io.EOF {
				vs.closed = true
				// clear all timeout
				vs.responseMapMutex.Lock()
				for _, cb := range vs.responseHdlMap {
					if cb.timeoutTimer != nil {
						cb.timeoutTimer.Stop()
					}
				}
				vs.responseMapMutex.Unlock()
				if vs.onCloseHdl != nil {
					vs.onCloseHdl(vs)
				}

				// exit for loop
				return
			} else if err != nil {
				VUtils.PrintError(err)
			}
		}
	}()

	return vs
}

// [msglen - 2bytes][encryptType - 1byte] (package ==>) [ck - 8bytes][isResponse - 1byte][requestId - 8bytes][operatorId - 3bytes][cmd - 2bytes][json]
func (vs *VSocket) loopParseMsg() {
	// first 2 bytes is uint16 represent the package size
	uint32Size := 4
	msglen := len(vs.msgBytes)
	if msglen > uint32Size {
		vs.contentSize = VUtils.BytesToUint32(&vs.msgBytes)
	}

	for vs.contentSize > 0 && msglen >= uint32Size+int(vs.contentSize) {

		encryptType := EncryptType(vs.msgBytes[uint32Size])
		packetBytes := vs.msgBytes[uint32Size+1 : uint32Size+int(vs.contentSize)]

		// it will override orizinal vsocket receiving buffer
		Decrypt(packetBytes, encryptType)

		receiveCK := VUtils.BytesToUint64(&packetBytes)

		if vs.syncCk && vs.receiveChecksum != receiveCK {
			fmt.Println("duplicate checksum. ", vs.receiveChecksum, receiveCK)
			//panic(errors.New("duplicate checksum. "))
			vs.isValid = false
			return
		}
		isResponse := packetBytes[8] > 0
		n := packetBytes[9:17]
		uuid := VUtils.BytesToUint64(&n)
		data := packetBytes[17:]
		// it is a callback's response
		// fmt.Println("isResponse === ", packetBytes[8])
		if isResponse {
			vs.responseMapMutex.Lock()
			cb, ok := vs.responseHdlMap[uuid]
			vs.responseMapMutex.Unlock()

			if ok {
				if cb.timeoutTimer != nil {
					cb.timeoutTimer.Stop()
				}
				if cb.succesHdl != nil {
					task := VSocketTask{cb: cb, requestId: uuid, bytes: &data}
					vs.taskList <- &task
				}

				vs.responseMapMutex.Lock()
				delete(vs.responseHdlMap, uuid)
				vs.responseMapMutex.Unlock()
			} else {
				fmt.Println("not found response for === ", uuid)
			}
		} else { // it is a normal incoming message
			if vs.onMsgHdl != nil {
				task := VSocketTask{cb: nil, requestId: uuid, bytes: &data}
				vs.taskList <- &task
			}
		}

		// reset vs ---
		vs.msgBytes = vs.msgBytes[uint32Size+int(vs.contentSize):]
		// update receiveChecksum
		vs.receiveChecksum++
		msglen = len(vs.msgBytes)

		if msglen > uint32Size {
			vs.contentSize = VUtils.BytesToUint32(&vs.msgBytes)
		} else {
			vs.contentSize = 0
		}
		// ------------
	}
}

/**
A--- send() ---> B
if B determine the message need a response by GAME LOGIC (not by protocol) ==> B --- response() ---> A, otherwise B do nothing
So: when A use send(), only sepecific a timeoutHdl() IF A make sure the message will be response() by B
*/

func (vs *VSocket) Send(bytes []byte, succesHdl func(vs *VSocket, requestId uint64, resData []byte), timeoutHdl func()) error {
	if vs.closed {
		return errors.New("socket is closed")
	}
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	// IMPORTANT: we use send checksum as requestid for callback determine
	requestId := vs.checksum

	callback := VSocketCallback{succesHdl: succesHdl}
	vs.responseMapMutex.Lock()
	vs.responseHdlMap[requestId] = &callback
	vs.responseMapMutex.Unlock()

	bt := VUtils.Uint64ToBytes(vs.checksum)
	isResponse := []byte{0}
	bt = append(bt, isResponse...)
	bt = append(bt, VUtils.Uint64ToBytes(requestId)...)
	bytes = append(bt, bytes...)

	Encrypt(bytes, vs.EncryptType)
	encryptType := []byte{byte(vs.EncryptType)}
	bytes = append(encryptType, bytes...)
	if len(bytes) > VSOCKET_MSG_MAX_LENGTH {
		msg := "ERROR: exceeded maximum message length"
		fmt.Println(msg)
		return errors.New(msg)
	}
	l := uint32(len(bytes))
	bytes = append(VUtils.Uint32ToBytes(l), bytes...)
	_, err := vs.conn.Write(bytes)
	if err != nil {
		VUtils.PrintError(err)
		vs.conn.Close()
	}
	if timeoutHdl != nil {
		callback.timeoutTimer = time.AfterFunc(VSOCKET_SEND_TIMEOUT, func() {
			timeoutHdl()
			vs.responseMapMutex.Lock()
			delete(vs.responseHdlMap, requestId)
			vs.responseMapMutex.Unlock()
		})
	}

	vs.checksum++
	return nil
}

func (vs *VSocket) Response(requestId uint64, bytes []byte) error {
	if vs.closed {
		return errors.New("socket is closed")
	}

	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	bt := VUtils.Uint64ToBytes(vs.checksum)
	isResponse := []byte{1}
	bt = append(bt, isResponse...)
	bt = append(bt, VUtils.Uint64ToBytes(requestId)...)
	bytes = append(bt, bytes...)

	Encrypt(bytes, vs.EncryptType)
	encryptType := []byte{byte(vs.EncryptType)}
	bytes = append(encryptType, bytes...)
	if len(bytes) > VSOCKET_MSG_MAX_LENGTH {
		msg := "ERROR: exceeded maximum message length"
		fmt.Println(msg)
		return errors.New(msg)
	}
	l := uint32(len(bytes))
	bytes = append(VUtils.Uint32ToBytes(l), bytes...)
	_, err := vs.conn.Write(bytes)
	if err != nil {
		VUtils.PrintError(err)
		vs.conn.Close()
	}
	vs.checksum++
	return nil
}

func (vs *VSocket) Close() {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()
	if vs.conn != nil {
		vs.conn.Close()
	}
}
