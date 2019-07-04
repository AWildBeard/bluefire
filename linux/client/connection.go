package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.ibm.com/mmitchell/ble/linux"

	"github.ibm.com/mmitchell/ble"
)

type Connections struct {
	lock        sync.RWMutex
	connections map[string]Connection
}

func NewConnections() Connections {
	return Connections{
		sync.RWMutex{},
		map[string]Connection{},
	}
}

func (cons *Connections) Lock() {
	cons.lock.Lock()
}

func (cons *Connections) Unlock() {
	cons.lock.Unlock()
}

func (cons *Connections) RLock() {
	cons.lock.RLock()
}

func (cons *Connections) RUnlock() {
	cons.lock.RUnlock()
}

func (cons *Connections) Connections() *map[string]Connection {
	return &cons.connections
}

// Connection holds a single BLE connection that is used to control and interact
// with remote hosts
type Connection struct {
	bleClient        ble.Client
	context          context.Context
	write            *ble.Characteristic
	writePipe        *io.PipeWriter
	read             *ble.Characteristic
	readPipe         *io.PipeReader
	readIndication   chan bool
	remoteIndication chan bool
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
		shutdown      = make(chan bool, 1)
		newCncl       func()
	)

	// Initiate the connection
	dlog.Printf("Dialing %v\n", addr.String())
	if newConnection.bleClient, err = dev.Dial(cntx, addr); err == nil {
		dlog.Printf("Connected to %v\n", addr.String())
	} else {
		dlog.Printf("Failed to connect to %v: %v\n", addr.String(), err)
		goto fail
	}

	// Must declare after a new connection has been created
	newCncl = func() {
		shutdown <- true
		newConnection.bleClient.ClearSubscriptions()
		newConnection.bleClient.CancelConnection()
		dlog.Println("BLE Connection Cancelled")
		cncl()
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

		// Bootstrap the indications and goroutine responsible from reading from
		// these now-verified attrs
		newConnection.readIndication = make(chan bool, 2)
		newConnection.remoteIndication = make(chan bool, 1)
		var (
			subscriptionHandler = func(rsp []byte) {
				dlog.Printf("Recieved indication from server: %v\n", rsp)
				newConnection.remoteIndication <- true
			}
		)

		dlog.Printf("Subscribing to %v\n", readUUID)
		if err = newConnection.bleClient.Subscribe(newConnection.read, true, subscriptionHandler); err != nil {
			dlog.Printf("Failed to subscribe to %v\n", readUUID)
			goto fail
		}
		dlog.Printf("Subscribed to %v\n", readUUID)

		dlog.Printf("Starting persistent reader thread")
		newConnection.readPipe, newConnection.writePipe = io.Pipe()

		go func(connection *Connection) {
			for true {
				select {
				// If context.Cancel() is called, this will shutdown
				// this goroutine so we aren't wasteful :D
				case <-shutdown:
					connection.writePipe.Close()
					return
				// Otherwise, handle remote indications and perform the read
				// operation that they are indicating
				case <-connection.remoteIndication:
					dlog.Printf("Recieved remote indication\n")
					dlog.Printf("Triggering 'starting' local indication\n")
					connection.readIndication <- true
					dlog.Printf("Triggered 'starting' local indication\n")
					var (
						bytes []byte
						err   error
					)

					dlog.Printf("Reading from the read characteristic: ")
					// While the remote has data for us to read, read and store that data to return later.
					for bytes, err = connection.bleClient.ReadCharacteristic(connection.read); err == nil && len(bytes) > 0; bytes, err = connection.bleClient.ReadCharacteristic(connection.read) {
						dlog.Printf("Writing %v bytes to pipe\n", len(bytes))
						connection.writePipe.Write(bytes)
						dlog.Printf("Wrote %v bytes to pipe\n", len(bytes))
					}

					dlog.Printf("Triggering 'finished' local indication\n")
					connection.readIndication <- true
					dlog.Printf("Triggered 'finished' local indication")

					// Write a zero length payload to prevent the reader from hanging
					// since we can't send io.EOF without closing the pipe, and
					// whats the point in closing the pipe if we just have to create
					// a new one for the next command?
					connection.writePipe.Write([]byte{})

					if err != nil {
						ErrChain <- Error{
							err,
							"iopipe",
						}
					}
				}
			}
		}(&newConnection)

		// Only then return a valid conneciton
		return newConnection, nil // This is our only valid return
	}

	err = fmt.Errorf("write and/or read characteristics not found")

fail:
	// Be sure to close the BLE connection on a bad exit
	newConnection.bleClient.CancelConnection()
	return Connection{}, err // This is a bad return
}

func (cntn Connection) WriteCommand(cmd string) error {
	var (
		err error
	)

	dlog.Printf("Writing %v to the write characteristic\n", cmd)
	if err = cntn.bleClient.WriteCharacteristic(cntn.write, []byte(cmd), false); err != nil {
		dlog.Printf("Failed to write %v to the write characteristic\n", cmd)
		return err
	}
	dlog.Printf("Wrote %v to the write characteristic\n", cmd)

	return err
}
