package main

import (
	"bytes"
	"context"
	"fmt"

	"github.ibm.com/mmitchell/ble/linux"

	"github.ibm.com/mmitchell/ble"
)

// Connection holds a single BLE connection that is used to control and interact
// with remote hosts
type Connection struct {
	bleClient ble.Client
	context   context.Context
	service   *ble.Characteristic
	write     *ble.Characteristic
	read      *ble.Characteristic
}

// NewConnection returns a new initialized connection that can be used immediately
// to send and recieve data.
func NewConnection(dev *linux.Device, addr ble.Addr) (Connection, error) {
	var (
		cntx, cncl    = context.WithCancel(context.Background())
		readSet       bool
		writeSet      bool
		newConnection = Connection{}
	)

	// Initiate the connection
	dlog.Printf("Dialing %v\n", addr.String())
	if newClient, err := dev.Dial(cntx, addr); err == nil {
		dlog.Printf("Connected to %v\n", addr.String())
		newConnection.bleClient = newClient
	} else {
		dlog.Printf("Failed to connect to %v: %v\n", addr.String(), err)
		return Connection{}, err
	}

	// TODO: Exchange MTU and enable DLE

	// Hunt for the service we need
	if services, err := newConnection.bleClient.DiscoverServices([]ble.UUID{serviceUUID}); err == nil {
		dlog.Println("Found service")
		if len(services) != 1 {
			newConnection.bleClient.CancelConnection()
			return Connection{}, fmt.Errorf("incorrect amount of services on remote host")
		}

		// Hunt for the characteristics we need
		if characteristics, err := newConnection.bleClient.DiscoverCharacteristics([]ble.UUID{writeUUID, readUUID}, services[0]); err == nil {
			dlog.Println("Found characteristics")
			if len(characteristics) != 2 {
				newConnection.bleClient.CancelConnection()
				return Connection{}, fmt.Errorf("incorrect amount of chracteristics on remote service")
			}

			// Make sure we find the correct characteristics
			for _, val := range characteristics {
				if val.UUID.Equal(writeUUID) {
					dlog.Println("Set write")
					writeSet = true
					newConnection.write = val
				} else if val.UUID.Equal(readUUID) {
					if (val.Property&ble.CharNotify) != 0 || (val.Property&ble.CharIndicate) != 0 {
						dlog.Println("Set read")
						readSet = true
						newConnection.read = val
					} else {
						return Connection{}, fmt.Errorf("read UUID does not support Notify or Indicate")
					}
				}
			}
		} else {
			// Close the BLE connection
			newConnection.bleClient.CancelConnection()
			return Connection{}, fmt.Errorf("%v", err)
		}
	} else {
		// Close the BLE connection
		newConnection.bleClient.CancelConnection()
		return Connection{}, fmt.Errorf("%v", err)
	}

	// Create a context.CancelFunc override to also close the BLE connection
	// when the context is canceled
	var newCncl = func() {
		newConnection.bleClient.CancelConnection()
		dlog.Println("BLE Connection Cancelled")
		cncl()
	}

	// Make sure we found both attrs
	if readSet && writeSet {
		cntx = context.WithValue(cntx, "cancel", context.CancelFunc(newCncl))
		newConnection.context = cntx
		// Only then return a valid conneciton
		return newConnection, nil
	}

	// Be sure to close the BLE connection on a bad exit
	newConnection.bleClient.CancelConnection()
	return Connection{}, fmt.Errorf("failed to set read and write")
}

func (cntn Connection) WriteCommand(cmd string) (string, error) {
	var (
		cmdReady            = make(chan bool, 1)
		subscriptionHandler = func(rsp []byte) {
			dlog.Printf("Recieved indication from server: %v\n", rsp)
			cmdReady <- true
		}
	)

	dlog.Printf("Subscribing to %v\n", readUUID)
	if err := cntn.bleClient.Subscribe(cntn.read, true, subscriptionHandler); err != nil {
		return "", err
	}
	dlog.Printf("Subscribed to %v\n", readUUID)

	// Be sure to unsubscribe
	defer func() {
		dlog.Printf("Unsubscribing from %v\n", readUUID)
		cntn.bleClient.ClearSubscriptions()
		dlog.Printf("Unsubscribed from %v\n", readUUID)
	}()

	dlog.Printf("Writing %v to the write characteristic\n", cmd)
	if err := cntn.bleClient.WriteCharacteristic(cntn.write, []byte(cmd), false); err != nil {
		return "", err
	}

	dlog.Printf("Awaiting indication from server")
	//time.Sleep(4 * time.Second) // Arbitrary until we sort out receiving data from the server
	<-cmdReady

	dlog.Printf("Reading from the read characteristic: ")
	var (
		buf   = bytes.NewBuffer(make([]byte, 0, 4096))
		bytes []byte
		err   error
	)

	// While the remote has data for us to read, read and store that data to return later.
	for bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read); err == nil && len(bytes) > 0; bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read) {
		dlog.Printf("Reading %v bytes\n", len(bytes))
		buf.Write(bytes)
	}

	if err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil
}
