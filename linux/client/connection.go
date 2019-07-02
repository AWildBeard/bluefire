package main

import (
	"context"
	"fmt"
	"io"

	"github.ibm.com/mmitchell/ble/linux"

	"github.ibm.com/mmitchell/ble"
)

// Connection holds a single BLE connection that is used to control and interact
// with remote hosts
type Connection struct {
	bleClient ble.Client
	context   context.Context
	write     *ble.Characteristic
	read      *ble.Characteristic
}

// NewConnection returns a new initialized connection that can be used immediately
// to send and recieve data.
func NewConnection(dev *linux.Device, addr ble.Addr) (Connection, error) {
	var (
		cntx, cncl    = context.WithCancel(context.Background())
		err           error
		readSet       bool
		writeSet      bool
		remoteMTU     int
		profile       *ble.Profile
		newConnection = Connection{}
		newCncl       = func() {
			newConnection.bleClient.CancelConnection()
			dlog.Println("BLE Connection Cancelled")
			cncl()
		}
	)

	// Initiate the connection
	dlog.Printf("Dialing %v\n", addr.String())
	if newConnection.bleClient, err = dev.Dial(cntx, addr); err == nil {
		dlog.Printf("Connected to %v\n", addr.String())
	} else {
		dlog.Printf("Failed to connect to %v: %v\n", addr.String(), err)
		goto fail
	}

	// Exchange MTU
	newConnection.bleClient.Conn().SetRxMTU(512)
	if remoteMTU, err = newConnection.bleClient.ExchangeMTU(512); err == nil {
		newConnection.bleClient.Conn().SetTxMTU(remoteMTU)
		dlog.Printf("The remote mtu is %v\n", remoteMTU-3)
	}

	// TODO: Enable DLE

	// Discover the remote profile
	if profile, err = newConnection.bleClient.DiscoverProfile(true); err == nil {
		// Hunt for the service we need
		for _, val := range profile.Services {
			if val.UUID.Equal(serviceUUID) {
				// Hunt for the characteristics we need
				for _, val := range val.Characteristics {
					if val.UUID.Equal(writeUUID) {
						dlog.Println("Set write")
						writeSet = true
						newConnection.write = val
					} else if val.UUID.Equal(readUUID) {
						if (val.Property&ble.CharNotify) != 0 || (val.Property&ble.CharIndicate) != 0 {
							dlog.Println("Set read")
							readSet = true
							newConnection.read = val
							dlog.Printf("Attribute %v CCCD: %v \n", val.UUID, val.UUID)
						} else {
							newConnection.bleClient.CancelConnection()
							dlog.Printf("%v does not allow Notify or Indicate\n", val.UUID)
							err = fmt.Errorf("read UUID does not support Notify or Indicate")
							goto fail
						}
					}
				}
			}
		}
	}

	// Make sure we found both attrs
	if readSet && writeSet {
		cntx = context.WithValue(cntx, "cancel", context.CancelFunc(newCncl))
		newConnection.context = cntx
		// Only then return a valid conneciton
		return newConnection, nil
	}

	err = fmt.Errorf("write and/or read characteristics not found")

fail:
	// Be sure to close the BLE connection on a bad exit
	newConnection.bleClient.CancelConnection()
	return Connection{}, err
}

func (cntn Connection) WriteCommand(cmd string) (*io.PipeReader, error) {
	var (
		cmdReady            = make(chan bool, 1)
		subscriptionHandler = func(rsp []byte) {
			dlog.Printf("Recieved indication from server: %v\n", rsp)
			cmdReady <- true
		}
	)

	dlog.Printf("Subscribing to %v\n", readUUID)
	if err := cntn.bleClient.Subscribe(cntn.read, true, subscriptionHandler); err != nil {
		return &io.PipeReader{}, err
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
		return &io.PipeReader{}, err
	}

	dlog.Printf("Awaiting indication from server")
	//time.Sleep(4 * time.Second) // Arbitrary until we sort out receiving data from the server
	<-cmdReady

	var reader, writer = io.Pipe()

	go func(writer *io.PipeWriter) {
		dlog.Printf("Reading from the read characteristic: ")
		var (
			bytes []byte
			err   error
		)

		// While the remote has data for us to read, read and store that data to return later.
		for bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read); err == nil && len(bytes) > 0; bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read) {
			dlog.Printf("Reading %v bytes\n", len(bytes))
			writer.Write(bytes)
		}

		err = writer.Close()

		if err != nil {
			ErrChain <- Error{
				err,
				"iopipe",
			}
		}

	}(writer)

	return reader, nil
}
