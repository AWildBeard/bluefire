package main

import (
	"bufio"
	"fmt"
)

func helpCmd(cmds []string) {
	if len(cmds) > 1 {
		printer.Printf("%v\n", commandInfo(cmds[1]))
	} else {
		printer.Printf("\033[2m\033[4mCommands: commands retrieve information\033[0m\n")
		fivePrint(validCommands())

		printer.Printf("\n\033[2m\033[4mActions: actions are commands that cause changes\033[0m\n")
		fivePrint(validActions())

		printer.Printf("\n\033[2m\033[4mUtilities: utilities are commands that help & control the interface\033[0m\n")
		fivePrint(validUtilities())
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

func connectCmd(controller *Controller, cmds []string) error {
	if len(cmds) < 2 {
		return fmt.Errorf("'connect' expects a target as a parameter")
	}

	if cmds[1][0] != '#' {
		return fmt.Errorf("Please select a target by its '#number'")
	}

	return controller.Connect(cmds[1])
}

func shellCmd(controller *Controller, stdinReader *bufio.Reader, stdoutWriter *bufio.Writer, cmds []string) error {
	var (
		shellID  = cmds[1]
		actionID = fmt.Sprintf("conn-%s", shellID)
	)

	if len(cmds) < 2 {
		return fmt.Errorf("'shell' expects a target as a parameter")
	}

	if cmds[1][0] != '#' {
		return fmt.Errorf("Please select a target by its '#number'")
	}

	if !controller.IsConnected(shellID) {
		return fmt.Errorf("Please connect to %s first using the 'connect' command", shellID)
	}

	controller.connections.RLock()
	// Wait for meeee!
	<-(*controller.connections.Connections())[actionID].Interact()
	controller.connections.RUnlock()

	// Wait until the user types exit before exiting the remote shell
	dlog.Printf("Exiting shell %v\n", shellID)
	return nil
}

func infoCmd(controller *Controller, cmds []string) error {
	if len(cmds) < 2 {
		return fmt.Errorf("'info' expects a target as a parameter")
	}

	if cmds[1][0] != '#' {
		return fmt.Errorf("Please select a target by its '#number'")
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
		return err
	}

	return nil
}
