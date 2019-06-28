package flag

import (
	"sync"
)

type Flag struct {
	mutex sync.RWMutex
	flag  bool
}

func NewFlag() Flag {
	return Flag{
		mutex: sync.RWMutex{},
		flag:  false,
	}
}

func (flag *Flag) Set() {
	flag.mutex.Lock()
	flag.flag = true
	flag.mutex.Unlock()
}

func (flag *Flag) Unset() {
	flag.mutex.Lock()
	flag.flag = false
	flag.mutex.Unlock()
}

func (flag *Flag) IsSet() bool {
	var fl bool
	flag.mutex.RLock()
	fl = flag.flag
	flag.mutex.RUnlock()

	return fl
}
