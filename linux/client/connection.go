package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.ibm.com/mmitchell/ble/linux/hci/cmd"

	"github.ibm.com/mmitchell/ble/linux"

	"github.ibm.com/mmitchell/ble"
)

// Connections act as a data holder to contain a unlimited
// amount of thread-safe connections. These are thread safe by
// having a RWMutex that is used before accessing this internal data
type Connections struct {
	lock        sync.RWMutex
	connections map[string]Connection
}

// NewConnections is a simple generator to create a Connections type
func NewConnections() Connections {
	return Connections{
		sync.RWMutex{},
		map[string]Connection{},
	}
}

// Lock is an override for the internal mutex's lock
func (cons *Connections) Lock() {
	cons.lock.Lock()
}

// Unlock is an override for the internal mutex
func (cons *Connections) Unlock() {
	cons.lock.Unlock()
}

// RLock is an override for the internal mutex
func (cons *Connections) RLock() {
	cons.lock.RLock()
}

// RUnlock is an override for the internal mutex
func (cons *Connections) RUnlock() {
	cons.lock.RUnlock()
}

// Connections returns a pointer to the internal data structure that
// holds Connections
func (cons *Connections) Connections() *map[string]Connection {
	return &cons.connections
}

// Connection holds a single BLE connection that is used to control and interact
// with remote hosts
type Connection struct {
	lock             *sync.RWMutex
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
		newConnection = Connection{lock: &sync.RWMutex{}}
		newCncl       func()
		dleReq        cmd.SetDataLengthCommand
		dleRsp        cmd.SetDataLengthCommandRP
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

	// Enable DLE
	dleReq = cmd.SetDataLengthCommand{
		ConnectionHandle: newConnection.bleClient.Conn().Handle(),
		TxOctets:         0xF8,
		TxTime:           0x4E2,
	}
	dleRsp = cmd.SetDataLengthCommandRP{}

	if err = dev.HCI.Send(&dleReq, &dleRsp); err != nil {
		dlog.Printf("failed to enable DLE: %v\n", err)
		dlog.Printf("HCI Response code: %v\n", dleRsp.Status)
		err = nil // This is a non-fatal error. Continue on.
	} else {
		dlog.Printf("Enabled DLE for L2CAP\n")
	}

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

	// Make sure we found both the read and write attrs
	if readSet && writeSet {
		// Override the cancel func to clean up subscriptions
		cntx = context.WithValue(cntx, "cancel", context.CancelFunc(newCncl))
		newConnection.context = cntx

		// Create the indications that will deal with remote indications from the server.
		newConnection.remoteIndication = make(chan bool, 1)
		var subscriptionHandler = func(rsp []byte) {
			dlog.Printf("Recieved indication from server: %v\n", rsp)
			select {
			case newConnection.remoteIndication <- true:
			default:
			}
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
	var (
		exitIndicate         = make(chan bool) // Signals the caller the threads exited
		internalExitIndicate = make(chan bool) // Signals the read thread to exit
		sendBuffer           = make(chan string)
		savedSttyState       string
	)
	// TODO: Investigate using a PTY library to control this stuff
	// instead of this stty madness

	// Save the TTY state
	if output, err := exec.Command("stty", "-F", "/dev/tty", "-g").Output(); err == nil {
		savedSttyState = string(output)

	} else {
		dlog.Printf("Failed to save stty state: %v\n", err)
		savedSttyState = "sane"
	}

	// Bootstrap the TTY to do the cool stuff
	// Disable character echoing (remote will handle it for us)
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	// Get rid of Ctrl-C interupts. Let remote handle it.
	exec.Command("stty", "-F", "/dev/tty", "intr", "undef").Run()
	// Get rid of Ctrl-Z interupts. Let remote handle it.
	exec.Command("stty", "-F", "/dev/tty", "susp", "undef").Run()

	// Handling reading
	go func() {
		var stdoutWriter = bufio.NewWriter(os.Stdout)
		for true {
			select {
			// Exit has been called, quit and cleanup
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
				cntn.lock.RLock()
				for bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read); err == nil && len(bytes) > 0 || len(bytes) == 1 && bytes[0] == 0; bytes, err = cntn.bleClient.ReadCharacteristic(cntn.read) {
					cntn.lock.RUnlock()
					dlog.Printf("Writing %v bytes to stdout\n", len(bytes))
					stdoutWriter.Write(bytes)
					stdoutWriter.Flush()
					dlog.Printf("Wrote %v bytes to stdout\n", len(bytes))
					cntn.lock.RLock()
				}
				cntn.lock.RUnlock()

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
			inputBuffer = make([]rune, 0, 25)
		)

		for {
			input, _, _ = stdinReader.ReadRune()
			switch input {
			// Ctrl-B has been pressed. This is our prefix key. Pressing q next will exit
			// the shell
			case 0x02:
				// If the prefix key was pressed, and the next key is not
				// a valid command, pass that key through to the remote shell
				if input, _, _ = stdinReader.ReadRune(); input != 'q' {
					goto fall
				}

				// Tell the caller that we done
				exitIndicate <- true

				// Tell our compatriot thread that we done
				internalExitIndicate <- true

				// Revert to a sane terminal environment using stty
				exec.Command("stty", "-F", "/dev/tty", "sane").Run()
				exec.Command("stty", "-F", "/dev/tty", savedSttyState).Run()
				return
			fall:
				fallthrough
			default:
				// Can't encue more data. Say so
				if len(inputBuffer)+1 >= cap(inputBuffer) {
					printer.Printf("Refusing to write data. Message cap is being reached.")
					printer.Printf("Consider exiting this session using Ctrl-b q and\nkilling the bluetooth connection with the kill command")
					continue
				}
				inputBuffer = append(inputBuffer, input)
				select {
				case sendBuffer <- string(inputBuffer[:len(inputBuffer)]):
					// Reslice. We don't care about stuff not getting GC'ed because
					// this is small data, and holding onto an extra 256 bytes
					// won't hurt anyone.
					//
					// By this point, we've sen't everything buffered. So clear the
					// buffer
					inputBuffer = inputBuffer[:0]
				default:
				}
			}
		}
	}()

	go func() {
		var cmd string
		for true {
			select {
			case cmd = <-sendBuffer:
				cntn.lock.Lock() // Write lock. Dissallows Read locks until Write lock is released
				// Send the typed command to the remote and get the response reader
				if err := cntn.WriteCommand(string(cmd)); err != nil {
					// Await the reader routine to tell us that it's recieved the indication
					// from the server that there is content to be read
					ErrChain <- Error{
						err,
						"write_handler",
					}
				}
				cntn.lock.Unlock()
			}
		}

	}()

	return exitIndicate
}

// WriteCommand takes the given string argument and writes
// it to the remote services 'write' characteristic. This 'write'
// characteristic is defined in the Bluetooth Connection Specification
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
