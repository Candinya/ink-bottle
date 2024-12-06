package main

import "time"

type cacheData struct {
	createdAt time.Time
	data      []byte
}
