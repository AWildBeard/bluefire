package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.ibm.com/mmitchell/ble"
	"github.ibm.com/mmitchell/ble/linux"
)

// Controller is the main gateway to interfacing and interacting with the ble
// library and contains a lot of our buisiness logic about how to control and
// interact with connect clients, etc.
type Controller struct {
	errChain chan Error
	bleDev   *linux.Device
	clients  map[string]Connection
	actions  Actions
	targets  Targets
}

// NewController returns an initialized controller ready to be accessed and modified
func NewController() *Controller {
	var dev *linux.Device
	dlog.Println("Accessing bluetooth device")

	// Access the linux bluetooth device. Pass our modified connection parameters
	// that specify some fine-tuned options.
	if newDev, err := linux.NewDevice(ble.OptConnParams(bleConnParams)); err == nil {
		dev = newDev
		ble.SetDefaultDevice(dev) // Make the libraries runtime happy :(
		dlog.Println("Successfully accessed bluetooth device", dev.HCI.Addr())
	} else {
		printer.Println("Failed to access bluetooth device")
		dlog.Printf("error: %v\n", err)
		printer.Println("Exiting")
		os.Exit(1)
	}

	// Declare the new controller. This is actually the controller that
	// will be returned to the caller
	var newController = Controller{
		errChain: make(chan Error, 10),
		bleDev:   dev,
		clients:  map[string]Connection{},
		actions:  NewActions(),
		targets:  NewTargets(),
	}

	// Pre-emptively start scanning to lessen the time for the caller to
	// get a viable list of targets
	dlog.Println("Starting scanning.")
	newController.ScanNow()

	dlog.Println("Starting error watcher.")
	// The error watcher is used to await for asynchronous errors from this
	// controller. Upon receiving them, the controller will clear a line of
	// output and write the error. It will then return the user to the main
	// prompt. Work still needs to be done to determine wheter this thread
	// should print prompt, if the main thread should recieve a notification
	// that prompt was just overwritten, etc.
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

// ScanNow begins the scanning process. This non-blocking function starts an
// asynchronous task.
func (cntrl *Controller) ScanNow() error {
	// Be sure that a scan isn't already running
	cntrl.actions.RLock()
	if _, ok := (*cntrl.actions.Actions())["scan"]; ok {
		cntrl.actions.RUnlock()
		return fmt.Errorf("already scanning")
	}
	cntrl.actions.RUnlock()

	// Grab a context to control the scan
	// TODO: Make fully cancelling context?
	var contxt, cancel = context.WithCancel(context.Background())
	var counter = 0

	// Make an advertisement handler. This is used to make sure that we
	// don't add a duplicate target, and to serialize the targets in a
	// user friendly way.
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

	// This filter is what is used to actually control what the user
	// can and can't see. This is done to control the targets list to
	// only show valid targets.
	var filter = func(a ble.Advertisement) bool {
		if a.Connectable() {
			for _, service := range a.Services() {
				if service.Equal(serviceUUID) {
					return true
				}
			}
		}

		return false
	}

	// Go ahead and add the scan to the list of actions. If scan actually fails
	// to start, it will be removed later :D
	cntrl.actions.Lock()
	(*cntrl.actions.Actions())["scan"] = context.WithValue(contxt, "cancel", cancel)
	cntrl.actions.Unlock()

	// Asyncronously start the scan
	go func() {
		if err := ble.Scan(contxt, false, handler, filter); err != nil {
			// If an error comes back, wrap it so we can serialize it,
			// and pass it to the controllers error watcher.
			cntrl.errChain <- Error{
				err,
				"scan",
			}
		}
	}()

	// No error :D
	// TODO: Change function signature?
	return nil
}

// RunningActions returns the list of running actions. This is used to monitor
// background tasks such as scanning and bluetooth connections.
func (cntrl *Controller) RunningActions() []string {
	cntrl.actions.RLock()
	var actions = cntrl.actions.Actions()
	var retKeys = make([]string, 0, len(*actions))

	// Copy
	for key := range *actions {
		retKeys = append(retKeys, key)
	}

	cntrl.actions.RUnlock()
	return retKeys
}

// CancelAction is used to kill background tasks and connections such as
// a ble scan or connection
func (cntrl *Controller) CancelAction(act string) error {
	cntrl.actions.Lock()
	var actions = cntrl.actions.Actions()
	// Determine if the action actually exists
	if cancel, ok := (*actions)[act]; ok {
		dlog.Printf("Action %v is running, canceling it.\n", act)
		// Call the actions CancelFunc. This can be used to trigger
		// conection closes and such.
		cancel.Value("cancel").(context.CancelFunc)()
		dlog.Printf("Canceled %v\n", act)

		// Remove the action
		delete(*actions, act)
		dlog.Printf("Removed %v from actions\n", act)

		cntrl.actions.Unlock()
		return nil
	}
	cntrl.actions.Unlock()

	// The action does not exist
	return fmt.Errorf("Could not find action %v", act)
}

// Targets returns a map of user friendly names mapped to
// the targets MAC address. This is used by the user to get a better
// feel of what targets are available to them to attack
func (cntrl *Controller) Targets() map[string]ble.Addr {
	var targets = map[string]ble.Addr{}

	cntrl.targets.RLock()
	for key, value := range *cntrl.targets.Targets() {
		targets[key] = value.Addr()
	}
	cntrl.targets.RUnlock()

	return targets
}

// DropTargets allows a caller to remove all the cached targets and start
// recapturing BLE advertisements. This is useful for mobile workstations
// that may move in and out of a targets range.
func (cntrl *Controller) DropTargets() {
	cntrl.actions.RLock()
	// If a scan is running, cancel it and retstart it. This is done to ensure
	// that the scan will return data on clients that are still in the area.
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

// Connect allows the caller to tell the Controller to connect to a BLE
// device by it's user friendly name.
func (cntrl *Controller) Connect(id string) error {
	var (
		err      error // zero value nil
		actionID = fmt.Sprintf("conn-%s", id)
	)

	// Make sure the Controller isn't already connected to the device
	if cntrl.IsConnected(id) {
		return fmt.Errorf("already connected to %s", id)
	}

	// Get the target's full ble advertisement, and use it's addr to attempt a connection
	if addr, err := cntrl.GetTarget(id); err == nil {
		if newClient, err := NewConnection(cntrl.bleDev, addr.Addr()); err == nil {
			dlog.Printf("Established connection to %s\n", id)
			cntrl.actions.Lock()
			// Connection successful, so store a cancel func to close the connection.
			(*cntrl.actions.Actions())[actionID] = newClient.context
			cntrl.actions.Unlock()

			// Store the client for the connection
			cntrl.clients[actionID] = newClient
			//run to retrieve cccd from server
			cntrl.clients[actionID].bleClient.DiscoverProfile(true)

		} else {
			return err
		}
	} else {
		return fmt.Errorf("%v is not a valid ID", id)
	}

	return err
}

// SendCommand allows a caller to cause a specific client to send a specific
// command. The function uses the user-friendly identifier to determine
// which client to send the command on.
func (cntrl *Controller) SendCommand(id, cmd string) (string, error) {
	var actionID = fmt.Sprintf("conn-%s", id)
	// Make sure the client is in fact connected before we attempt to send
	// the command
	if cntrl.IsConnected(id) {
		return cntrl.clients[actionID].WriteCommand(cmd)
	}

	return "", fmt.Errorf("connection id %v not found", id)
}

// IsConnected allows a caller to determine if a user-friendly ID is tied to an
// active connection.
func (cntrl *Controller) IsConnected(id string) bool {
	var actionID = fmt.Sprintf("conn-%s", id)

	_, ok := cntrl.clients[actionID]

	return ok
}

// GetTarget is used to retrieve the full BLE advertisement used to connect
// to the broadcaster of the advertisment. It uses the user-friendly ID
// to retrieve the correct target's advertisement.
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
