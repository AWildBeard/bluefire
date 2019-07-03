package main

import (
	"bufio"
	"fmt"
	"strings"
)

func helpCmd(cmds []string) {
	if len(cmds) > 1 {
		printer.Printf("%v\n", CommandInfo(cmds[1]))
	} else {
		printer.Printf("\033[2m\033[4mCommands: commands retrieve information\033[0m\n")
		fivePrint(ValidCommands())

		printer.Printf("\n\033[2m\033[4mActions: actions are commands that cause changes\033[0m\n")
		fivePrint(ValidActions())

		printer.Printf("\n\033[2m\033[4mUtilities: utilities are commands that help & control the interface\033[0m\n")
		fivePrint(ValidUtilities())
	}
}

func targetsCmd(controller *Controller) {
	var mp = controller.Targets()
	if len(mp) == 0 {
		printer.Println("Nothing here.")
		return
	}

	var counter = 1
	for key, value := range mp {
		printer.Printf("\033[1m%-3v\033[0m : \033[2m%v\033[0m", key, value)

		if counter == 2 {
			fmt.Println()
			counter = 1
			continue
		} else if counter == 1 {
			fmt.Printf("   ")
		}

		counter++
	}

	if counter != 1 {
		fmt.Println()
	}
}

func connectCmd(controller *Controller, cmds []string) {
	if len(cmds) < 2 {
		printer.Println("'connect' expects a target as a parameter")
		return
	}

	if cmds[1][0] != '#' {
		printer.Println("Please select a target by its '#number'")
		return
	}

	if err := controller.Connect(cmds[1]); err != nil {
		printer.Printf("%v\n", err)
	}
}

func shellCmd(controller *Controller, stdinReader *bufio.Reader, cmds []string) {
	var (
		shellID = cmds[1]
		input   string
	)

	if len(cmds) < 2 {
		printer.Println("'shell' expects a target as a parameter")
		return
	}

	if cmds[1][0] != '#' {
		printer.Println("Please select a target by its '#number'")
		return
	}

	if !controller.IsConnected(shellID) {
		printer.Printf("Please connect to %s first using the 'connect' command\n", shellID)
		return
	}

	// Wait until the user types exit before exiting the remote shell
	for input != "exit" {
		remoteShellPrompt(shellID)
		input, _ = stdinReader.ReadString('\n')
		input = strings.TrimRight(input, "\r\n")
		switch input {
		case "exit":
			break
		default:
			// Send the typed command to the remote and get the response
			if reader, indications, err := controller.SendCommand(shellID, input); err == nil {
				dlog.Printf("Awaiting local indication")
				if <-*indications {
					dlog.Printf("Received indication for reading")
					var (
						buf       = make([]byte, 1024)
						err       error
						bytesRead int
						exit      bool
					)

					for !exit {
						select {
						case <-*indications:
							dlog.Printf("Recieved exit indicationn")
							exit = true
						default:
							for bytesRead, err = reader.Read(buf); err == nil && bytesRead > 0; bytesRead, err = reader.Read(buf) {
								fmt.Printf("%s", buf)
								for i := range buf {
									buf[i] = 0
								}
							}
						}
					}
				}
			} else {
				printer.Printf("%v\n", err)
			}
		}
	}

	printer.Printf("Exiting shell %s\n", shellID)
}

func infoCmd(controller *Controller, cmds []string) {
	if len(cmds) < 2 {
		printer.Println("'info' expects a target as a parameter")
		return
	}

	if cmds[1][0] != '#' {
		printer.Println("Please select a target by its '#number'")
		return
	}

	// Get the target the user wants info for
	if target, err := controller.GetTarget(cmds[1]); err == nil {
		// Print the infor about the target
		printer.Printf("Target information for: %v\n", target.Addr())
		printer.Printf("\tRSSI: %v\n", target.RSSI())

		printer.Printf("\tTxPower: %vdbm\n", target.TxPowerLevel())

		if services := target.Services(); len(services) > 0 {
			printer.Printf("\tServices:\n")
			for _, service := range services {
				printer.Printf("\t\t%v\n", service)
			}
		} else {
			printer.Printf("\t%v is not advertising services\n", target.Addr())
		}
	} else {
		printer.Printf("%v\n", err)
	}
}
