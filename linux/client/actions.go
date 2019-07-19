package main

import (
	"context"
	"sync"
)

// Actions allows the library to store a map of which actions are running in the
// background and has the ability to cancel those background actions.
type Actions struct {
	lock    sync.RWMutex
	actions map[string]context.Context
}

// NewActions returns a newly initialized Actions struct that is ready to rock.
func NewActions() Actions {
	return Actions{
		sync.RWMutex{},
		map[string]context.Context{},
	}
}

// RLock is an override for Actions mutex
func (action *Actions) RLock() {
	action.lock.RLock()
}

// RUnlock is an override for Actions mutex
func (action *Actions) RUnlock() {
	action.lock.RUnlock()
}

// Lock is an override for Actions mutex
func (action *Actions) Lock() {
	action.lock.Lock()
}

// Unlock is an override for Actions mutex
func (action *Actions) Unlock() {
	action.lock.Unlock()
}

// Actions returns a pointer to the internal data structure
// used by actions for action management
func (action *Actions) Actions() *map[string]context.Context {
	return &action.actions
}
