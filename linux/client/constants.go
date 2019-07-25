package main

import (
	"github.com/AWildBeard/ble"
	"github.com/AWildBeard/ble/linux/hci/cmd"
)

var (
	bleConnParams = cmd.LECreateConnection{
		LEScanInterval:        0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		LEScanWindow:          0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		InitiatorFilterPolicy: 0x00,      // White list is not used
		PeerAddressType:       0x00,      // Public Device Address
		PeerAddress:           [6]byte{}, //
		OwnAddressType:        0x00,      // Public Device Address
		ConnIntervalMin:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnIntervalMax:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnLatency:           0x0000,    // 0x0000 - 0x01F3; N * 1.25 msec
		SupervisionTimeout:    0x0048,    // 0x000A - 0x0C80; N * 10 msec
		MinimumCELength:       0xFFFF,    // 0x0000 - 0xFFFF; N * 0.625 msec
		MaximumCELength:       0xFFFF,    // 0x0000 - 0xFFFF; N * 0.625 msec
	}
	serviceUUID = ble.MustParse("10a47006-0001-4c30-a9b7-ca7d92240018")
	writeUUID   = ble.MustParse("10a47006-0002-4c30-a9b7-ca7d92240018")
	readUUID    = ble.MustParse("10a47006-0003-4c30-a9b7-ca7d92240018")
)
