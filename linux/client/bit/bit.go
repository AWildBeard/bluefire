package bit

import (
	"sync"
)

type Bit struct {
	mutex sync.RWMutex
	bit   bool
}

func NewBit() Bit {
	return Bit{
		mutex: sync.RWMutex{},
		bit:   false,
	}
}

func (bit *Bit) Set() {
	bit.mutex.Lock()
	bit.bit = true
	bit.mutex.Unlock()
}

func (bit *Bit) Unset() {
	bit.mutex.Lock()
	bit.bit = false
	bit.mutex.Unlock()
}

func (bit *Bit) IsSet() bool {
	var fl bool
	bit.mutex.RLock()
	fl = bit.bit
	bit.mutex.RUnlock()

	return fl
}
