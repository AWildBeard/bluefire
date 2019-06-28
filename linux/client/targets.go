package main

import (
	"sync"

	"github.ibm.com/mmitchell/ble"
)

type Targets struct {
	targets map[string]ble.Addr
	lock    sync.RWMutex
}

func NewTargets() Targets {
	return Targets{
		targets: map[string]ble.Addr{},
		lock:    sync.RWMutex{},
	}
}

func (targets *Targets) DropTargets() {
	targets.lock.Lock()
	targets.targets = map[string]ble.Addr{}
	targets.lock.Unlock()
}

func (targets *Targets) RLock() {
	targets.lock.RLock()
}

func (targets *Targets) RUnlock() {
	targets.lock.RUnlock()
}

func (targets *Targets) Lock() {
	targets.lock.Lock()
}

func (targets *Targets) Unlock() {
	targets.lock.Unlock()
}

func (targets *Targets) Targets() map[string]ble.Addr {
	return targets.targets
}
