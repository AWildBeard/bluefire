package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
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
	read             *ble.Characteristic
	exitIndication   chan bool // This is a blocking signal
	remoteIndication chan bool // This is a signal triggered by receiving a indication
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

		// Create the indications that will deal with remote indications from the server.
		newConnection.remoteIndication = make(chan bool, 1)
		var subscriptionHandler = func(rsp []byte) {
			dlog.Printf("Recieved indication from server: %v\n", rsp)
			newConnection.remoteIndication <- true
		}

		dlog.Printf("Subscribing to %v\n", readUUID)
		if err = newConnection.bleClient.Subscribe(newConnection.read, true, subscriptionHandler); err != nil {
			dlog.Printf("Failed to subscribe to %v\n", readUUID)
			goto fail
		}
		dlog.Printf("Subscribed to %v\n", readUUID)

		// Only then return a valid conneciton
		return newConnection, nil // This is our only valid return
	}

	err = fmt.Errorf("write and/or read characteristics not found")

fail:
	// Be sure to close the BLE connection on a bad exit
	newConnection.bleClient.CancelConnection()
	return Connection{}, err // This is a bad return
}

// Interact is called to start two threads that will take control
// of stdin and stdout. They will tell the client when to return to normal
// operation by using the returned channel
func (cntn Connection) Interact() chan bool {
	var exitIndicate = make(chan bool)
	var internalExitIndicate = make(chan bool)

	// Disable input buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()

	// Disable character echoing (remote will handle it for us)
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	// Handling reading
	go func() {
		var stdoutWriter = bufio.NewWriter(os.Stdout)
		for true {
			select {
			case <-internalExitIndicate:
				return
			// Otherwise, handle remote indications and perform the read
			// operation that they are indicating
			case <-cntn.remoteIndication:
				dlog.Printf("Recieved remote indication\n")
				var (
					bytes []byte
					err   error
				)

				dlog.Printf("Reading from the read characteristic: ")
				// While the remote has data for us to read, read and store that data to return later.
				for bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read); err == nil && len(bytes) > 0; bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read) {
					if len(bytes) == 1 && bytes[0] == 0 {
						break
					}

					dlog.Printf("Writing %v bytes to stdout\n", len(bytes))
					stdoutWriter.Write(bytes)
					stdoutWriter.Flush()
					dlog.Printf("Wrote %v bytes to stdout\n", len(bytes))
				}

				if err != nil {
					ErrChain <- Error{
						err,
						"read_handler",
					}
				}
			}
		}

	}()

	// Handling writing
	go func() {
		var (
			stdinReader = bufio.NewReader(os.Stdin)
			input       rune
		)

		for {
			input, _, _ = stdinReader.ReadRune()
			switch input {
			case 0x02:
				if input, _, _ = stdinReader.ReadRune(); input != 'q' {
					goto fall
				}

				// Tell the caller that we done
				exitIndicate <- true

				// Tell our compatriot thread that we done
				internalExitIndicate <- true

				exec.Command("stty", "-F", "/dev/tty", "sane").Run()
				return
			fall:
				fallthrough
			default:
				// Send the typed command to the remote and get the response reader
				if err := cntn.WriteCommand(string(input)); err != nil {
					// Await the reader routine to tell us that it's recieved the indication
					// from the server that there is content to be read
					ErrChain <- Error{
						err,
						"write_handler",
					}
				}
			}
		}
	}()

	return exitIndicate
}

func (cntn Connection) WriteCommand(cmd string) error {
	var (
		err error
	)

	dlog.Printf("Writing %v to the write characteristic\n", cmd)
	if err = cntn.bleClient.WriteCharacteristic(cntn.write, []byte(cmd), true); err != nil {
		dlog.Printf("Failed to write %v to the write characteristic\n", cmd)
		return err
	}
	dlog.Printf("Wrote %v to the write characteristic\n", cmd)

	return err
}
