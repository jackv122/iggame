package com

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
	"unsafe"
)

type VUtilsT struct {
}

var VUtils = &VUtilsT{}

func (*VUtilsT) PrintError(err error) {
	panic(err)
	//println("Error ======= ", err.Error())
}

type TimeKeeper struct {
	Timer *time.Timer
}

// delay: in second
func (*VUtilsT) RepeatCall(callback func(delay float64), delay float64, repeatTime int, keeper *TimeKeeper) {
	if keeper == nil {
		keeper = &TimeKeeper{}
	}
	count := 0
	var f func() = nil
	timeSecond := float64(time.Second)
	duration := delay * timeSecond
	f = func() {
		callback(delay)
		if repeatTime > 0 {
			count++
			if count < repeatTime {
				keeper.Timer = time.AfterFunc(time.Duration(duration), f)
			}
		} else {
			keeper.Timer = time.AfterFunc(time.Duration(duration), f)
		}
	}
	keeper.Timer = time.AfterFunc(time.Duration(duration), f)
}

func (*VUtilsT) HashAmount(userId uint64, amt Amount, currency Currency) string {
	// fmt.Println("hash ", userId, amt, currency)
	uidStr := string(VUtils.Uint64ToBytes(userId))
	amtstr := string(VUtils.Uint64ToBytes(uint64(amt)))
	text := uidStr + "_" + amtstr + "_" + string(currency) + HASH_KEY
	hash := sha256.Sum256([]byte(text))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (*VUtilsT) HashString(str string) string {
	text := str + "_" + HASH_KEY
	hash := sha256.Sum256([]byte(text))
	//fmt.Println("HashString " + str + "   " + base64.StdEncoding.EncodeToString(hash[:]))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (*VUtilsT) GetRandInt(i int) int {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return r.Intn(i)
}

func (*VUtilsT) GetRandFloat64() float64 {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return r.Float64()
}

func (*VUtilsT) Uint64ToBytes(i uint64) []byte {
	bytes := *(*[8]byte)(unsafe.Pointer(&i))
	return bytes[:]
}

func (*VUtilsT) BytesToUint64(bytes *[]byte) uint64 {
	intBytes := [8]byte((*bytes)[:8])
	return *(*uint64)(unsafe.Pointer(&intBytes))
}

func (*VUtilsT) Uint16ToBytes(i uint16) []byte {
	bytes := *(*[2]byte)(unsafe.Pointer(&i))
	return bytes[:]
}

func (*VUtilsT) BytesToUint16(bytes *[]byte) uint16 {
	intBytes := [2]byte((*bytes)[:2])
	return *(*uint16)(unsafe.Pointer(&intBytes))
}

func (*VUtilsT) Uint32ToBytes(i uint32) []byte {
	bytes := *(*[4]byte)(unsafe.Pointer(&i))
	return bytes[:]
}

func (*VUtilsT) BytesToUint32(bytes *[]byte) uint32 {
	intBytes := [4]byte((*bytes)[:4])
	return *(*uint32)(unsafe.Pointer(&intBytes))
}

func (*VUtilsT) WalletLocalMessage(operatorId OperatorID, cmd uint16, obj any) []byte {
	bytes, _ := json.Marshal(obj)
	pres := VUtils.Uint16ToBytes(cmd)
	bytes = append(pres, bytes...)

	pres = []byte(operatorId)
	bytes = append(pres, bytes...)
	return bytes
}

func (*VUtilsT) WalletLocalMessageString(operatorId OperatorID, cmd uint16, str string) []byte {
	bytes := []byte(str)
	pres := VUtils.Uint16ToBytes(cmd)
	bytes = append(pres, bytes...)

	pres = []byte(operatorId)
	bytes = append(pres, bytes...)
	return bytes
}

func (*VUtilsT) WalletLocalMessageUint64(operatorId OperatorID, cmd uint16, n uint64) []byte {
	bytes := VUtils.Uint64ToBytes(n)
	pres := VUtils.Uint16ToBytes(cmd)
	bytes = append(pres, bytes...)

	pres = []byte(operatorId)
	bytes = append(pres, bytes...)
	return bytes
}

// global method --
func RemoveElementFromArray[T comparable](arr []T, v T) []T {
	for i := 0; i < len(arr); i++ {
		if v == arr[i] {
			arr[i] = arr[len(arr)-1]
			return arr[:len(arr)-1]
		}
	}
	return arr
}

func truncateAmount(amount Amount) Amount {
	// truncate
	amountVal := float64(amount)
	amountVal = float64(uint64(amountVal*100)) / 100.0
	return Amount(amountVal)
}

func FormatAmount(amount Amount) string {
	// truncate
	amountVal := float64(amount)
	amountVal = float64(uint64(amountVal*100)) / 100.0
	str := fmt.Sprintf("%.2f", amountVal)
	return str
}

func getRoomKey(operatorId OperatorID, roomId RoomId) string {
	return string(operatorId) + string(roomId)
}
