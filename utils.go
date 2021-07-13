package main

import (
	"os"
	"time"
)

func defaultEnv(key, defval string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}

	return defval
}

func unixNowMs() int64 {
	return time.Now().UTC().UnixNano() / 1000000
}
