package main

import (
	"bufio"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/AWildBeard/ble"
)

// ShellServer is the logical type that allows a client to read and execute
// programs.
type ShellServer struct {
	shell            *exec.Cmd
	outputReader     *bufio.Reader
	outRdrLock       sync.Mutex
	inputWriter      *bufio.Writer
	inWtrLock        sync.Mutex
	lastTimeRead     time.Time
	lastTimeNotified time.Time
	subscribed       bool
	clientIndicate   chan bool
	dataIndicate     chan bool
	resumeDataChecks chan bool
}

// NewShellServer creates a new ShellServer
func NewShellServer() *ShellServer {
	var newServer = &ShellServer{
		clientIndicate:   make(chan bool, 1),
		dataIndicate:     make(chan bool),
		resumeDataChecks: make(chan bool, 1),
		outRdrLock:       sync.Mutex{},
		inWtrLock:        sync.Mutex{},
	}

	// Bootstrap the shell process and initialize our ShellServer
	newServer.shell = exec.Command("bash", "-l")
	var outputPipe, _ = newServer.shell.StdoutPipe()
	var inputPipe, _ = newServer.shell.StdinPipe()
	newServer.outputReader = bufio.NewReader(outputPipe)
	newServer.inputWriter = bufio.NewWriter(inputPipe)

	// If we are unable to start the shell, this is a fatal error
	// and we need to exit
	if err := newServer.shell.Start(); err != nil {
		ilog.Fatalf("Failed to start shell: %v\n", err)
		os.Exit(1)
	}

	// Wait before bootstrapping the shell
	time.Sleep(1 * time.Second)
	inputPipe.Write([]byte("python -c 'import pty;pty.spawn(\"/bin/bash\")'\n"))
	time.Sleep(1 * time.Second)

	return newServer
}

// ServeWrite allows the ShellServer to take in commands to execute
func (srv *ShellServer) ServeWrite(req ble.Request, rsp ble.ResponseWriter) {
	dlog.Printf("Got a write from %v\n", req.Conn().RemoteAddr())

	// Can't give us commands to execute if you are not subscribed to
	// recieve the output of those commands
	if !srv.subscribed {
		dlog.Printf("Responding NOT SUBSCRIBED!\n")
		rsp.SetStatus(ble.ErrAuthentication)
		return
	}

	srv.inWtrLock.Lock()
	srv.inputWriter.Write(req.Data())
	srv.inputWriter.Flush() // Does not execute command
	srv.inWtrLock.Unlock()

	// Be quick. This pre-emtively notifies the client to retrieve data
	// from the server. This is good because of terminal echo and tab-auto
	// complete.
	srv.notifyClient()
}

// ServeRead allows the Bluetooth Low Energy client to retrieve the output
// of whatever command they wrote to Stdin
func (srv *ShellServer) ServeRead(req ble.Request, rsp ble.ResponseWriter) {
	dlog.Printf("Got a read from %v\n", req.Conn().RemoteAddr())

	// Can't give us commands to execute if you are not subscribed to
	// recieve the output of those commands
	if !srv.subscribed {
		dlog.Printf("Responding NOT SUBSCRIBED!\n")
		rsp.SetStatus(ble.ErrAuthentication)
		return
	}

	// TODO: Copy goroutine instead?
	var (
		buf []byte
		err error
	)

	select {
	// Our data watcher has confirmed that there is data to be read
	case <-srv.dataIndicate:
		srv.outRdrLock.Lock()
		// This will tell the thread that buffers data for us to
		// continue. By having the thread wait for this indication,
		// we are guarenteed the outRdrLock
		srv.resumeDataChecks <- true

		var (
			bytesBuffered = srv.outputReader.Buffered()
			cap           = rsp.Cap()
		)

		if bytesBuffered == 0 {
			// Why did we recieve a dataIndicate then? Eh, better safe than sorry.
			// Otherwise we block on the other cases below and dats bad.
			dlog.Printf("No bytes buffered for reading, responding with zero len")
			buf = []byte{}
		} else if cap > bytesBuffered { // Only send what needs to be sent
			buf = make([]byte, bytesBuffered)
			_, err = srv.outputReader.Read(buf)
		} else { // Only send what needs to be sent
			buf = make([]byte, cap)
			_, err = srv.outputReader.Read(buf)
		}

		srv.outRdrLock.Unlock()
	default:
		dlog.Printf("No data ready according to dataIndicate. Responding with zero len")
		buf = []byte{}
	}

	if err != nil {
		dlog.Printf("Caught error in reading: %v\n", err)
	}

	// Finally write the data to the client
	dlog.Printf("Responding to client\n")
	if _, err := rsp.Write(buf); err != nil {
		ilog.Printf("Failed to write data to client: %v\n", err)
	}
	dlog.Printf("Responded to client\n")

	srv.lastTimeRead = time.Now()
}

// ServeNotify allows the server to indicate to the client when there
// is data to be read from stdout.
func (srv *ShellServer) ServeNotify(req ble.Request, n ble.Notifier) {
	dlog.Printf("Recieved subscription from %v\n", req.Conn().RemoteAddr())

	// This channel is used to indicate our data watcher to exit and cleanup
	var exitChan = make(chan bool, 1)

	// Don't allow more that one subscription!
	if srv.subscribed {
		dlog.Printf("we already have a subscription, ignoring!\n")
		n.Close()
		return
	}

	srv.subscribed = true

	// This small goroutine listens for activity on the stdout pipe and
	// will cause an indication to the subscribed client. This tells the client
	// that there is content to read :D
	go func(srv *ShellServer) {
		var kill = make(chan bool)
		for {
			select {
			// The signal to control exiting this goroutine
			case <-exitChan:
				// Indicate to a subroutine that it too needs to exit
				kill <- true
				return
			default:
				srv.outRdrLock.Lock()
				// Calling peek is kindof shitty, but we neeed the underlying reader
				// To buffer data and thats how we trigger it. Through testing, it buffers
				// all of the data that needs to be read. Subsequent peeks in ServeRead
				// cause even more buffering which is what we want.
				if _, err := srv.outputReader.Peek(1); err == nil {
					srv.outRdrLock.Unlock()
					dlog.Printf("Found data to read!")
					// Only indicate if it's been a while since the client
					// has read from us
					var (
						localExitChan = make(chan bool, 1)
					)

					// This thread only exists to send multiple notifies,
					// in the case that the client doesn't recieve the first for whatever reason.
					// This will continue to indicate the client until they read from the read
					// attribute.
					go func() {
						var currentTime time.Time

						for {
							currentTime = time.Now()
							select {
							// The client read data, so exit!
							case <-localExitChan:
								return
							// The client unsubscribed, so exit!
							case <-kill:
								return
							// Indicate the client only when necessary
							default:
								var (
									sinceLastRead   = currentTime.Sub(srv.lastTimeRead)
									sinceLastNotify = currentTime.Sub(srv.lastTimeNotified)
								)

								if sinceLastRead > 250*time.Millisecond && sinceLastNotify > 250*time.Millisecond {
									dlog.Printf("Found data that has not been read yet!")
									srv.notifyClient()
								} else if sinceLastRead < sinceLastNotify {
									// The sinceLastRead counter has to go the farthest to get to 2 seconds
									time.Sleep((250 * time.Millisecond) - sinceLastRead)
								} else {
									// The sinceLastNotify counter has to go the farthest to get to 2 seconds
									time.Sleep((250 * time.Millisecond) - sinceLastNotify)
								}
							}
						}
					}()

					// This will block. We do this so that the thread above can indicate
					// the client until it reads. When it reads, this will unblock
					srv.dataIndicate <- true

					// This will not block. Just to signal the thread above to exit
					localExitChan <- true

					// Wait for the main reader thread to acquire it's lock just in case.
					<-srv.resumeDataChecks
				} else {
					dlog.Printf("Caught error while peeking for data: %v\n", err)
				}
			}
		}
	}(srv)

	var data = []byte{}

	for {
		select {
		// The client unsubscribed
		case <-n.Context().Done():
			dlog.Println("Client unsubscribed")
			srv.subscribed = false
			exitChan <- true
			return
		// Our data wathcer threads have told us to indicate the client
		case <-srv.clientIndicate:
			dlog.Println("Notifying the client")

			// Trigger the notification
			if _, err := n.Write(data); err != nil {
				ilog.Printf("Error during notify: %v\n", err)
			}
		}
	}
}

// This funcition serves as a utility to track when notifications occur in addition
// to triggering a notification to the client.
func (srv *ShellServer) notifyClient() {
	srv.clientIndicate <- true
	srv.lastTimeNotified = time.Now()
}
