package main

import (
	"bytes"
	"context"
	"fmt"

	"github.ibm.com/mmitchell/ble"
	"github.ibm.com/mmitchell/ble/linux"
)

//changed one char from server data test, changed 3-4th msb to 1
const btServiceUUID = "00000001-0000-1000-8000-00805F9B34FC"
const btCharacteristicStdinUUID = "00000002-0000-1000-8000-00805F9B34FC"
const btCharacteristicStdoutUUID = "00000003-0000-1000-8000-00805F9B34FC"

//take that cs professors
var StdinCharacteristic *ble.Characteristic
var StdoutCharacteristic *ble.Characteristic
var ShellService *ble.Service

func main() {
	fmt.Printf("Server Test\n")
	var dev *linux.Device

	if newDev, err := linux.NewDeviceWithName("Bose-QC40"); err == nil {
		dev = newDev
	} else {
		fmt.Printf("Failed to attach HCI dev: %v \n", err)
	}

	ble.SetDefaultDevice(dev)

	//Initialize characteristics, UUIDs
	var btStdinServer = &Stdin{}
	var btStdoutServer = &Stdout{}
	var serverUUID = "10a47006-0001-4c30-a9b7-ca7d92240018"
	var btCharacteristicStdinUUID = "10a47006-0002-4c30-a9b7-ca7d92240018"
	var btCharacteristicStdoutUUID = "10a47006-0003-4c30-a9b7-ca7d92240018"

	//initialize the LE service
	//make the data buffers

	//create stdout
	btStdoutServer.c = make(chan struct{}, 1)
	btStdoutServer.Data = bytes.NewBuffer(make([]byte, 511))
	StdoutCharacteristic = newStdout(btStdoutServer, btCharacteristicStdoutUUID)

	//create stdin
	btStdinServer.Data = bytes.NewBuffer(make([]byte, 511))
	btStdinServer.stdout = btStdoutServer
	StdinCharacteristic = newStdin(btStdinServer, btCharacteristicStdinUUID)

	ShellService = ble.NewService(ble.MustParse(serverUUID))
	ShellService.AddCharacteristic(StdinCharacteristic)
	ShellService.AddCharacteristic(StdoutCharacteristic)
	if err := ble.AddService(ShellService); err != nil {
		fmt.Printf("Error adding service %v: %v", ShellService, err)
	}
	//WHOMST THIS DO, FIND OUT
	var context = ble.WithSigHandler(context.WithCancel(context.Background()))
	ble.AdvertiseNameAndServices(context, "Bose-QC40", ShellService.UUID)

}

//deal with input from client
func newStdin(handler *Stdin, UUID string) *ble.Characteristic {
	var characteristic = ble.NewCharacteristic(ble.MustParse(UUID))
	characteristic.HandleWrite(handler)
	return characteristic
}

//stdout characteristic: writing to the standard out when command completes.
func newStdout(handler *Stdout, UUID string) *ble.Characteristic {
	var characteristic = ble.NewCharacteristic(ble.MustParse(UUID))
	characteristic.HandleRead(handler)
	characteristic.HandleIndicate(handler)
	return characteristic
}
