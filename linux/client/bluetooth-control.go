package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.ibm.com/mmitchell/ble"
	"github.ibm.com/mmitchell/ble/linux"
	"github.ibm.com/mmitchell/bluefire/linux/client/flag"
)

type Controller struct {
	ready   flag.Flag
	bleDev  *linux.Device
	actions map[string]context.Context
	targets Targets
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
		ready:   flag.NewFlag(),
		bleDev:  dev,
		actions: map[string]context.Context{},
		targets: NewTargets(),
	}

	newController.ScanNow()

	return &newController
}

func (cntrl *Controller) IsReady() bool {
	return cntrl.ready.IsSet()
}

func (cntrl *Controller) ScanNow() error {
	if _, ok := cntrl.actions["scan"]; ok {
		return fmt.Errorf("Already scanning!")
	}

	var contxt, cancel = context.WithCancel(context.Background())
	var counter = 0

	var handler = func(a ble.Advertisement) {
		cntrl.targets.Lock()
		var targets = cntrl.targets.Targets()

		for _, val := range targets {
			if val.String() == a.Addr().String() {
				cntrl.targets.Unlock()
				return
			}
		}

		counter++
		cntrl.targets.Targets()[fmt.Sprintf("#%v", counter)] = a.Addr()
		cntrl.targets.Unlock()
	}

	var filter = func(a ble.Advertisement) bool {
		if a.Connectable() /*&& a.LocalName() == "echo-server"*/ {
			return true
		}

		return false
	}

	go ble.Scan(ble.WithSigHandler(contxt, cancel), false, handler, filter)

	cntrl.actions["scan"] = context.WithValue(contxt, "cancel", cancel)
	return nil
}

func (cntrl *Controller) RunningActions() []string {
	var retKeys = make([]string, 0, len(cntrl.actions))

	for key := range cntrl.actions {
		retKeys = append(retKeys, key)
	}

	return retKeys
}

func (cntrl *Controller) CancelAction(act string) error {
	if action, ok := cntrl.actions[act]; ok {
		action.Value("cancel").(context.CancelFunc)()
		delete(cntrl.actions, act)
		return nil
	}

	return fmt.Errorf("Could not find action %v", act)
}

func (cntrl *Controller) Targets() map[string]ble.Addr {
	var targets = map[string]ble.Addr{}

	cntrl.targets.RLock()
	for key, value := range cntrl.targets.Targets() {
		targets[key] = value
	}
	cntrl.targets.RUnlock()

	return targets
}

func (cntrl *Controller) DropTargets() {
	if _, ok := cntrl.actions["scan"]; ok {
		cntrl.CancelAction("scan")
		cntrl.targets.DropTargets()
		time.Sleep(500 * time.Millisecond) // No spam :D
		cntrl.ScanNow()
		return
	}

	cntrl.targets.DropTargets()
}

func ValidActions() []string {
	return []string{"help", "scan", "cancel [action]",
		"ps", "targets", "purge-targets"}
}
