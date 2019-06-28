package main

import (
	"context"
	"sync"
)

type Actions struct {
	lock    sync.RWMutex
	actions map[string]context.Context
}

func NewActions() Actions {
	return Actions{
		sync.RWMutex{},
		map[string]context.Context{},
	}
}

func (action *Actions) RLock() {
	action.lock.RLock()
}

func (action *Actions) RUnlock() {
	action.lock.RUnlock()
}

func (action *Actions) Lock() {
	action.lock.Lock()
}

func (action *Actions) Unlock() {
	action.lock.Unlock()
}

func (action *Actions) Actions() *map[string]context.Context {
	return &action.actions
}
