package main

import (
	"bytes"
	"sync"
)

type OutputData struct {
	data *bytes.Buffer
	lock sync.RWMutex
}

func (data *OutputData) Lock() {
	data.lock.Lock()
}

func (data *OutputData) Unlock() {
	data.lock.Unlock()
}

func (data *OutputData) RLock() {
	data.lock.RLock()
}

func (data *OutputData) RUnlock() {
	data.lock.RUnlock()
}

func (data *OutputData) Data() *bytes.Buffer {
	return data.data
}
