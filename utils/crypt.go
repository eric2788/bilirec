package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomHexString(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func RandomHexStringMust(n int) string {
	s, err := RandomHexString(n)
	if err != nil {
		panic(err)
	}
	return s
}
