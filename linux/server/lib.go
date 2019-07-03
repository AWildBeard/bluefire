package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.ibm.com/mmitchell/ble"
)

//generate structs for stdin and sttdout
type Stdout struct {
	Data *bytes.Buffer
	c    chan struct{}
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
	go runCommandAsync(command, srv.stdout.Data, srv.stdout.c)
}

//server write -> client
func (srv *Stdout) ServeRead(req ble.Request, rsp ble.ResponseWriter) {
	fmt.Printf("Got a read from %v\n", req.Conn().RemoteAddr())
	//fmt.Printf("Writing %s\n", srv.Data.Bytes())
	//only write rsp.Cap() of data at a time, client just calls several times
	if _, err := rsp.Write(srv.Data.Next(rsp.Cap())); err != nil {
		fmt.Printf("Failed to write data to client: %v\n", err)
	}

}

//notify client on finished shell command
func (srv *Stdout) ServeNotify(req ble.Request, n ble.Notifier) {
	fmt.Printf("Notification Subscribed\n")
	data := []byte("done")
	for {
		select {
		case <-n.Context().Done():
			return
		case <-srv.c:
			if _, err := n.Write(data); err != nil {
				fmt.Printf("Error during notify: %v\n", err)
			}
		}
	}
}

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
	cmd.Wait()
	//stdout.ServeIndicate()
}

func runCommandAsync(command string, stdoutData *bytes.Buffer, c chan struct{}) {
	stdoutData.Reset()
	var cmdName = strings.Split(command, " ")
	var cmd *exec.Cmd
	if len(cmdName) > 1 {
		cmd = exec.Command(cmdName[0], cmdName[1:]...)
	} else {
		cmd = exec.Command(cmdName[0])
	}
	stdout, _ := cmd.StdoutPipe()
	r := bufio.NewReader(stdout)
	cmd.Start()
	goUpdate(r, stdoutData, cmd.ProcessState.ExitCode, c)
	cmd.Wait()
}

func goUpdate(r *bufio.Reader, stDoutData *bytes.Buffer, getExit func() int, c chan struct{}) {
	var firstData bool = false
	for {
		data, _, err := r.ReadLine()
		var dataStr string
		dataStr = fmt.Sprintf("%s\n", string(data))
		if getExit() >= 0 || err != nil {
			break
		}
		if err != io.EOF {
			if firstData == false {
				firstData = true
				c <- struct{}{}
			}
			stDoutData.Write([]byte(dataStr))
		}
	}
}
