package com

import (
	"encoding/base64"
	"os"
)

var encryptKey = [100]uint8{253, 173, 190, 183, 230, 61, 54, 164, 248, 201, 72, 170, 223, 32, 103, 190, 46, 118, 63, 209, 231, 161, 162, 11, 46, 161, 155, 88, 250, 12, 68, 207, 12, 54, 228, 146, 130, 113, 204, 180, 151, 50, 207, 49, 245, 2, 4, 60, 89, 223, 164, 138, 17, 114, 105, 150, 181, 228, 157, 15, 181, 58, 214, 132, 217, 192, 244, 39, 86, 56, 189, 151, 9, 147, 10, 36, 104, 15, 171, 232, 116, 33, 163, 203, 67, 252, 57, 168, 38, 94, 143, 12, 231, 161, 57, 192, 91, 13, 219, 237}

func EncryptForGame(bytes []byte) {
	// swap
	for i := 0; i < len(bytes); i++ {
		var pre int = i / 2
		t := bytes[i]
		bytes[i] = bytes[pre]
		bytes[pre] = t
	}
	keyLen := len(encryptKey)
	for i := 0; i < len(bytes); i++ {
		bytes[i] ^= encryptKey[i%keyLen]
	}
}

func DecryptForGame(bytes []byte) {
	keyLen := len(encryptKey)
	for i := 0; i < len(bytes); i++ {
		bytes[i] ^= encryptKey[i%keyLen]
	}

	// swap
	for i := len(bytes) - 1; i >= 0; i-- {
		var pre int = i / 2
		t := bytes[i]
		bytes[i] = bytes[pre]
		bytes[pre] = t
	}
}

func EncryptForProxy(bytes []byte) {
}

func DecryptForProxy(bytes []byte) {
}

func Encrypt(bytes []byte, encryptType EncryptType) {
	switch encryptType {
	case PROXY_ENCRYPT:
		EncryptForProxy(bytes)
	case GAME_ENCRYPT:
		EncryptForGame(bytes)
	}
}

func Decrypt(bytes []byte, encryptType EncryptType) {
	switch encryptType {
	case PROXY_ENCRYPT:
		DecryptForProxy(bytes)
	case GAME_ENCRYPT:
		DecryptForGame(bytes)
	}
}

func EncryptBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func DecryptBase64(content string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(content)
}

func EncryptToFile(url string, bytes []byte, encryptType EncryptType) error {

	Encrypt(bytes, 0)
	err := os.WriteFile(url, bytes, 0644)
	return err
}

func DecryptFromFile(url string, encryptType EncryptType) ([]byte, error) {

	bytes, err := os.ReadFile(url)
	if err != nil || len(bytes) == 0 {
		return nil, err
	}

	Decrypt(bytes, encryptType)
	return bytes, nil
}
