package main

/*
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
*/

/*
func runCommandAsync(command string, stdout *Stdout, c chan struct{}) {
	stdout.data.Lock()

	var (
		data    = stdout.data.Data()
		cmdName = strings.Split(command, " ")
		cmd     *exec.Cmd
	)

	data.Reset()

	if len(cmdName) > 1 {
		cmd = exec.Command(cmdName[0], cmdName[1:]...)
	} else {
		cmd = exec.Command(cmdName[0])
	}

	var (
		stdoutPipe, _ = cmd.StdoutPipe()
		stdoutReader  = bufio.NewReaderSize(stdoutPipe, 1024)
		finishSignal  = make(chan bool, 1)
	)

	cmd.Start()
	go func() {
		if _, err := stdoutReader.Peek(1); err == nil {
			stdout.c <- struct{}{}
		} else {
			return
		}

		var buf = make([]byte, 512)

		for {
			select {
			case <-finishSignal:
				return
			default:
				if _, err := stdoutReader.Read(buf); err == nil {
					data.Write(buf)
				}
			}
		}
	}()

	cmd.Wait()
	finishSignal <- true

	// Signal our reader routine that we are done :D
	stdout.data.Unlock()
}

/*
func goUpdate(stdoutReader *bufio.Reader, stDoutData *bytes.Buffer, getExit func() int, c chan struct{}) {
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
*/
