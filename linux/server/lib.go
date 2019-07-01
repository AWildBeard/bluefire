package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.ibm.com/mmitchell/ble"
)

//generate structs for stdin and sttdout
type Stdout struct {
	Data     *bytes.Buffer
	//Position int
}
type Stdin struct {
	Data   *bytes.Buffer
	stdout *Stdout
}

//client write -> server
func (srv *Stdin) ServeWrite(req ble.Request, rsp ble.ResponseWriter) {
	var command = string(req.Data())
	fmt.Printf("Running command %s\n", command)
	runCommand(command, srv.stdout.Data)
}
//server write -> client
func (srv *Stdout) ServeRead(req ble.Request, rsp ble.ResponseWriter) {
	fmt.Printf("Got a read from %v\n", req.Conn().RemoteAddr())
	fmt.Printf("Writing %s\n", srv.Data.Bytes())
	//only write rsp.Cap() of data at a time, client just calls several times
	if _, err := rsp.Write(srv.Data.Next(rsp.Cap())); err != nil {
		fmt.Printf("Failed to write data to client: %v\n", err)
	}

}
//this code is under construction
/*func (srv *Stdout) ServeIndicate(req Request, n Notifier) {
	fmt.Printf("Indicate\n")
}*/

//run the command
func runCommand(command string, stdoutData *bytes.Buffer) {
	stdoutData.Reset()
	var cmdName = strings.Split(command, " ")
	var cmd *exec.Cmd
	if len(cmdName) > 1 {
		cmd = exec.Command(cmdName[0], cmdName[1:]...)
	} else {
		cmd = exec.Command(cmdName[0])
	}
	if ret, err := cmd.Output(); err == nil {
		fmt.Printf("%s\n", ret)
		if _, err := stdoutData.Write(ret); err != nil {
			fmt.Printf("Read error: %v\n", err)
		}
	} else {
		fmt.Printf("Output error: %v\n", err)
	}
}
