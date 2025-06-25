package test

import (
	"math/rand"
	"time"
)

func ParsePointersToString(s *string) string {
	if s == nil {
		return ""
	} else {
		return *s
	}
}

func GetRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
