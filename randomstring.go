package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var mutex sync.Mutex

func int63() int64 {
	mutex.Lock()
	v := src.Int63()
	mutex.Unlock()
	return v
}

var src = rand.NewSource(time.Now().UnixNano())

// RandString - Generate a random string of n characters
func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// RandomIC - generate a valid but random IC number
func RandomIC() string {
	rand.Seed(time.Now().UnixNano())
	year := rand.Intn(58) + 41
	month := rand.Intn(12) + 1
	day := rand.Intn(28) + 1
	state := rand.Intn(5) + 1
	stuff := rand.Intn(9999) + 1
	return fmt.Sprintf("%02d%02d%02d%02d%04d", year, month, day, state, stuff)
}
