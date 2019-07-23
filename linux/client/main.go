package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	debug        bool
	printVersion bool
	printer      *message.Printer
	dlog         *log.Logger
	buildversion string
	buildmode    string
	release      = "beta"
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&printVersion, "version", false, "Output version info an exit")
	printer = message.NewPrinter(language.English)
}

func main() {
	flag.Parse()

	if debug {
		dlog = log.New(os.Stdout, "DBG: ", 0)
	} else {
		dlog = log.New(ioutil.Discard, "", 0)
	}

	defer dlog.Println("Exiting program.")

	if printVersion {
		dlog.Println("Printing version info and exiting.")
		printer.Printf("%s-%s-%s\n", release, buildmode, buildversion)
		return
	}

	printer.Println("\033[38;5;160m                       (                    		")
	printer.Println("   (   (               )\033[38;5;214m\\\033[38;5;160m )                 	")
	printer.Println(" ( )\033[38;5;214m\\\033[38;5;160m  )\033[38;5;214m\\\033[38;5;160m   (     (   (()\033[38;5;214m/\033[38;5;160m(  (   (      (   		")
	printer.Println(" )((_)((_) ))\033[38;5;214m\\\033[38;5;160m   ))\033[38;5;214m\\\033[38;5;160m   \033[38;5;214m/\033[38;5;160m(_)) )\033[38;5;214m\\\033[38;5;160m  )(    ))\033[38;5;214m\\\033[38;5;160m  	")
	printer.Println("((\033[38;5;20m_\033[38;5;160m)\033[38;5;20m_  _  \033[38;5;214m/\033[38;5;160m((\033[38;5;226m_\033[38;5;160m) \033[38;5;214m/\033[38;5;160m((\033[38;5;226m_\033[38;5;160m) (\033[38;5;20m_\033[38;5;160m))\033[38;5;20m_\033[38;5;214m|\033[38;5;160m((\033[38;5;226m_\033[38;5;160m)(()\033[38;5;214m\\  /\033[38;5;160m((\033[38;5;226m_\033[38;5;160m) 	\033[38;5;20m")
	printer.Println(" | _ )| |\033[38;5;160m(\033[38;5;20m_\033[38;5;160m))( (\033[38;5;20m_\033[38;5;160m))   \033[38;5;20m| |_   \033[38;5;160m(\033[38;5;20m_\033[38;5;160m) ((\033[38;5;20m_\033[38;5;160m)(\033[38;5;20m_\033[38;5;160m))   	\033[38;5;20m")
	printer.Println(" | _ \\| || || |/ -_)  | __|  | || '_|/ -_)  	")
	printer.Println(" |___/|_| \\_,_|\\___|  |_|    |_||_|  \\___|  	")
	printer.Println("\033[m")
	printer.Println("\033[4m\033[1mUse 'help' to see a list of help topics.\033[m")
	printer.Println()

	dlog.Println("Creating bluetooth controller")
	var (
		controller   = NewController()
		stdinReader  = bufio.NewReader(os.Stdin)
		stdoutWriter = bufio.NewWriter(os.Stdout)
	)

	dlog.Println("Starting Bluetooth Control Loop.")

	var (
		cmdHistory  = make([]string, 0, 256)
		input       = make([]byte, 0, 256)
		key         byte
		firstCombo  bool
		secondCombo bool
		cursorPos   int
		historyPos  int
	)

	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()

	for true {
		prompt()

		for key, _ = stdinReader.ReadByte(); key != '\n'; key, _ = stdinReader.ReadByte() {
			dlog.Printf("0x%X pressed\n", key)
			switch key {
			case 0x7F:
				if cursorPos-1 >= 0 {
					printer.Printf("\0337")
					prompt()
					cursorPos--
					if cursorPos+1 <= len(input) {
						input = append(input[:cursorPos], input[cursorPos+1:]...)
					} else {
						input = input[:cursorPos]
					}
					printer.Printf("%s", input)
					printer.Printf("\0338")
					printer.Printf("\033[3D")
				} else {
					prompt()
				}
				continue
			case 0x09:
				if cursorPos >= 1 {
					var output = string(input)
					for key := range commandDescriptions {
						if strings.HasPrefix(key, string(input)) {
							output = key
							cursorPos = len(key)
							input = []byte(key)
							break
						}
					}

					prompt()
					printer.Printf(output)
				} else {
					prompt()
				}
				continue
			case 0x1B:
				firstCombo = true
				continue
			case 0x5B:
				secondCombo = true
				continue
			case 0x43: // Right arrow
				if firstCombo && secondCombo && cursorPos+1 <= len(input) {
					firstCombo = false
					secondCombo = false
					printer.Printf("\0337")
					prompt()
					printer.Printf("%s", input)
					printer.Printf("\0338")
					printer.Printf("\033[3D")
					cursorPos++
				} else {
					dlog.Printf("Not moving cursor")
					printer.Printf("\0337")
					prompt()
					printer.Printf("%s", input)
					printer.Printf("\0338")
					printer.Printf("\033[4D")
				}
				continue
			case 0x44: // Left arrow
				if firstCombo && secondCombo && cursorPos-1 >= 0 {
					firstCombo = false
					secondCombo = false
					printer.Printf("\0337")
					prompt()
					printer.Printf("%s", input)
					printer.Printf("\0338")
					printer.Printf("\033[5D")
					cursorPos--
				} else {
					dlog.Printf("Not moving cursor")
					printer.Printf("\0337")
					prompt()
					printer.Printf("%s", input)
					printer.Printf("\0338")
					printer.Printf("\033[4D")
				}
				continue
			case 0x41: // Up arrow
				prompt()
				if firstCombo && secondCombo && historyPos-1 >= 0 {
					firstCombo = false
					secondCombo = false
					historyPos--
					printer.Printf("%s", cmdHistory[historyPos])
					input = []byte(cmdHistory[historyPos])
					cursorPos = len(input)
				} else {
					printer.Printf("%s", input)
				}
				continue
			case 0x42: // Down arrow
				prompt()
				if firstCombo && secondCombo && historyPos+1 <= len(cmdHistory)-1 {
					firstCombo = false
					secondCombo = false
					historyPos++
					printer.Printf("%s", cmdHistory[historyPos])
					input = []byte(cmdHistory[historyPos])
					cursorPos = len(input)
				} else {
					printer.Printf("%s", input)
				}
				continue
			}

			input = append(input, key)
			cursorPos++
		}

		// Reset the cursor position
		cursorPos = 0

		// Convert the input into the command
		var command = string(input[:len(input)])

		// Update the command history
		if len(cmdHistory) == 256 {
			cmdHistory = cmdHistory[1:]
		} else {
			historyPos++
		}
		cmdHistory = append(cmdHistory, command)

		// Clear the input buffer
		input = input[:0]

		var cmds = strings.Split(command, " ")

		dlog.Printf("Command %v\n", cmds[0])
		switch cmds[0] {
		case "?":
			fallthrough
		case "h":
			fallthrough
		case "help":
			helpCmd(cmds)
		case "scan":
			if err := controller.ScanNow(); err != nil {
				printer.Printf("%v\n", err)
			}
		case "ps":
			fivePrint(controller.RunningActions())
		case "kill":
			if len(cmds) < 2 {
				printer.Println("'kill' expects an action as a second argument")
				continue
			}

			var err = controller.CancelAction(cmds[1])
			if err != nil {
				printer.Printf("%v\n", err)
			}

		case "targets":
			targetsCmd(controller)
		case "purge-targets":
			controller.DropTargets()
		case "connect":
			fallthrough
		case "shell":
			var err = connectCmd(controller, cmds)
			if err != nil {
				printer.Printf("%v\n", err)
				continue
			}

			err = shellCmd(controller, stdinReader, stdoutWriter, cmds)
			if err != nil {
				printer.Printf("%v\n", err)
			}
			exec.Command("stty", "-F", "/dev/tty", "sane").Run()
			exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
		case "info":
			var err = infoCmd(controller, cmds)
			if err != nil {
				printer.Printf("%v\n", err)
			}
		case "exit":
			return
		case "cls":
			fallthrough
		case "clear":
			clearScreen()
			prompt()
		default:
			printer.Println("Please enter a valid command. Type 'help' for a list of commands")
		}
	}
}

func clearScreen() {
	fmt.Printf("\033[2J\033[f")
}

func prompt() {
	fmt.Printf("\033[2K\r")
	printer.Print("\033[38;5;21mbf\033[38;5;196m>\033[m ")
}

func fivePrint(words []string) {
	if len(words) == 0 {
		printer.Println("Nothing here.")
		return
	}

	for i, word := range words {
		if (i+1)%5 == 0 {
			printer.Printf("'%s'\n", word)
		} else {
			printer.Printf("'%s'   ", word)
		}
	}

	if len(words)%5 != 0 {
		fmt.Println()
	}
}
