package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/AWildBeard/ble"
	"github.com/AWildBeard/ble/linux"
)

//changed one char from server data test, changed 3-4th msb to 1
const (
	btServiceUUID              = "10a47006-0001-4c30-a9b7-ca7d92240018"
	btCharacteristicStdinUUID  = "10a47006-0002-4c30-a9b7-ca7d92240018"
	btCharacteristicStdoutUUID = "10a47006-0003-4c30-a9b7-ca7d92240018"
)

var (
	debug        bool
	version      bool
	buildversion string
	buildmode    string
	release      = "beta"
	dlog         *log.Logger
	ilog         *log.Logger
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Enables debug output for this program")
	flag.BoolVar(&version, "version", false, "Print version information and exit")
}

func main() {
	flag.Parse()

	ilog = log.New(os.Stdout, "", 0)

	if version {
		ilog.Printf("%s-%s-%s\n", release, buildmode, buildversion)
		return
	}

	if debug {
		dlog = log.New(os.Stderr, "", 0)
	} else {
		dlog = log.New(ioutil.Discard, "", 0)
	}

	var (
		dev       *linux.Device
		bleServer *ble.Service

		// Initialize our concepts of stdout and stdin for bluetooth
		btShellServer        = NewShellServer()
		stdoutCharacteristic *ble.Characteristic
		stdinCharacteristic  *ble.Characteristic

		err error
	)

	if dev, err = linux.NewDeviceWithName("Bose-QC40"); err != nil {
		ilog.Printf("Failed to attach HCI dev: %v\n", err)
		return
	}
	dlog.Printf("Accessed HCI dev: %v\n", dev.HCI.Addr())

	ble.SetDefaultDevice(dev)

	//create stdout
	stdoutCharacteristic = ble.NewCharacteristic(ble.MustParse(btCharacteristicStdoutUUID))
	stdoutCharacteristic.HandleRead(btShellServer)
	stdoutCharacteristic.ReadHandler = btShellServer
	stdoutCharacteristic.HandleIndicate(btShellServer)

	//create stdin
	stdinCharacteristic = ble.NewCharacteristic(ble.MustParse(btCharacteristicStdinUUID))
	stdinCharacteristic.HandleWrite(btShellServer)
	stdinCharacteristic.WriteHandler = btShellServer

	bleServer = ble.NewService(ble.MustParse(btServiceUUID))
	bleServer.AddCharacteristic(stdinCharacteristic)
	bleServer.AddCharacteristic(stdoutCharacteristic)

	if err := ble.AddService(bleServer); err != nil {
		ilog.Printf("Error adding service %v: %v", bleServer, err)
	}

	// Create the context to cancel the server see "context" docs
	var context = ble.WithSigHandler(context.WithCancel(context.Background()))

	dlog.Printf("Hosting the server and listening for connections!")

	// Advertise the service and it's characteristics
	ble.AdvertiseNameAndServices(context, "Bose-QC40", bleServer.UUID)
	dlog.Printf("Exiting")
}
