package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.ibm.com/mmitchell/ble"
	"github.ibm.com/mmitchell/ble/linux"
)

type Controller struct {
	errChain chan Error
	bleDev   *linux.Device
	actions  Actions
	targets  Targets
}

func NewController() *Controller {
	var dev *linux.Device
	dlog.Println("Accessing bluetooth device")

	if newDev, err := linux.NewDevice(ble.OptConnParams(bleConnParams)); err == nil {
		dev = newDev
		ble.SetDefaultDevice(dev)
		dlog.Println("Successfully accessed bluetooth device", dev.HCI.Addr())
	} else {
		printer.Println("Failed to access bluetooth device")
		dlog.Printf("error: %v\n", err)
		printer.Println("Exiting")
		os.Exit(1)
	}

	var newController = Controller{
		errChain: make(chan Error, 10),
		bleDev:   dev,
		actions:  NewActions(),
		targets:  NewTargets(),
	}

	dlog.Println("Starting scanning.")
	newController.ScanNow()

	dlog.Println("Starting error watcher.")
	go func() {
		for true {
			select {
			case err := <-newController.errChain:
				if fmt.Sprintf("%v", err.err) == "context canceled" {
					continue
				}

				newController.CancelAction(err.source)

				// Clear line and print error
				printer.Printf("\033[2K\r%v\n", err)
				prompt()
			default:
				time.Sleep(250 * time.Millisecond)
			}
		}
	}()

	return &newController
}

func (cntrl *Controller) ScanNow() error {
	cntrl.actions.RLock()
	if _, ok := (*cntrl.actions.Actions())["scan"]; ok {
		cntrl.actions.RUnlock()
		return fmt.Errorf("Already scanning!")
	}
	cntrl.actions.RUnlock()

	var contxt, cancel = context.WithCancel(context.Background())
	var counter = 0

	var handler = func(a ble.Advertisement) {
		cntrl.targets.Lock()
		var targets = cntrl.targets.Targets()

		for _, val := range *targets {
			if val.Addr().String() == a.Addr().String() {
				cntrl.targets.Unlock()
				return
			}
		}

		counter++
		(*cntrl.targets.Targets())[fmt.Sprintf("#%v", counter)] = a
		cntrl.targets.Unlock()
	}

	var filter = func(a ble.Advertisement) bool {
		if a.Connectable() /*&& a.LocalName() == "echo-server"*/ {
			return true
		}

		return false
	}

	go func() {
		if err := ble.Scan(contxt, false, handler, filter); err != nil {
			cntrl.errChain <- Error{
				err,
				"scan",
			}
		}
	}()

	// Wait for thread startup, and an error, if it arises.

	cntrl.actions.Lock()
	(*cntrl.actions.Actions())["scan"] = context.WithValue(contxt, "cancel", cancel)
	cntrl.actions.Unlock()
	return nil
}

func (cntrl *Controller) RunningActions() []string {
	cntrl.actions.RLock()
	var actions = cntrl.actions.Actions()
	var retKeys = make([]string, 0, len(*actions))

	for key := range *actions {
		retKeys = append(retKeys, key)
	}

	cntrl.actions.RUnlock()
	return retKeys
}

func (cntrl *Controller) CancelAction(act string) error {
	cntrl.actions.Lock()
	var actions = cntrl.actions.Actions()
	if cancel, ok := (*actions)[act]; ok {
		dlog.Printf("Action %v is running, canceling it.\n", act)
		cancel.Value("cancel").(context.CancelFunc)()
		dlog.Printf("Canceled %v\n", act)
		delete(*actions, act)
		dlog.Printf("Removed %v from actions\n", act)
		cntrl.actions.Unlock()
		return nil
	}
	cntrl.actions.Unlock()

	return fmt.Errorf("Could not find action %v", act)
}

func (cntrl *Controller) Targets() map[string]ble.Addr {
	var targets = map[string]ble.Addr{}

	cntrl.targets.RLock()
	for key, value := range *cntrl.targets.Targets() {
		targets[key] = value.Addr()
	}
	cntrl.targets.RUnlock()

	return targets
}

func (cntrl *Controller) DropTargets() {
	cntrl.actions.RLock()
	if _, ok := (*cntrl.actions.Actions())["scan"]; ok {
		// This allows DropTargets() to re-populate with data.
		dlog.Println("A scan is already happening. Stopping and restarting scan.")
		cntrl.actions.RUnlock()
		cntrl.CancelAction("scan")
		cntrl.targets.DropTargets()
		time.Sleep(500 * time.Millisecond) // No spam :D
		cntrl.ScanNow()
		return
	}
	cntrl.actions.RUnlock()

	dlog.Println("Dropping targets. No scan is running.")
	cntrl.targets.DropTargets()
}

func (cntrl *Controller) GetTarget(key string) (ble.Advertisement, error) {
	var target ble.Advertisement
	cntrl.targets.RLock()
	if newTarget, ok := (*cntrl.targets.Targets())[key]; ok {
		target = newTarget
	} else {
		cntrl.targets.RUnlock()
		return nil, fmt.Errorf("target %v does not exist", key)
	}
	cntrl.targets.RUnlock()

	return target, nil
}
