package main

import (
	"sync"

	"github.com/AWildBeard/ble"
)

// Targets is a data structure to hold our scan data
type Targets struct {
	// Targets are defined by our scan results
	targets map[string]ble.Advertisement
	lock    sync.RWMutex
}

// NewTargets is a generator function to create a Targets
func NewTargets() Targets {
	return Targets{
		targets: map[string]ble.Advertisement{},
		lock:    sync.RWMutex{},
	}
}

// DropTargets flushes all the targets in the data store
func (targets *Targets) DropTargets() {
	targets.lock.Lock()
	targets.targets = map[string]ble.Advertisement{}
	targets.lock.Unlock()
}

// RLock is an override for Targets mutex
func (targets *Targets) RLock() {
	targets.lock.RLock()
}

// RUnlock is an override for Targets mutex
func (targets *Targets) RUnlock() {
	targets.lock.RUnlock()
}

// Lock is an override for Targets mutex
func (targets *Targets) Lock() {
	targets.lock.Lock()
}

// Unlock is an override for Targets mutex
func (targets *Targets) Unlock() {
	targets.lock.Unlock()
}

// Targets returns a pointer to the internal data structure
// that holds 'targets'
func (targets *Targets) Targets() *map[string]ble.Advertisement {
	return &targets.targets
}
