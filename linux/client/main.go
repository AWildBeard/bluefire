package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.ibm.com/mmitchell/bluefire/linux/client/bit"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	debug        bool
	printVersion bool
	printer      *message.Printer
	dlog         *log.Logger
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
		printer.Println("alpha")
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
	printer.Println("\033[0m")
	printer.Println("\033[4m\033[1mUse 'help' to see a list of help topics.\033[m")
	printer.Println()

	dlog.Println("Creating bluetooth controller")
	var (
		controller   = NewController()
		interupts    = make(chan os.Signal, 1)
		stdinReader  = bufio.NewReader(os.Stdin)
		stdoutWriter = bufio.NewWriter(os.Stdout)
		cmdRunning   = bit.NewBit()
		currentShell = ""
		inShell      = bit.NewBit()
	)

	dlog.Println("Starting interupt handler")
	signal.Notify(interupts, os.Interrupt)
	go func() {
		dlog.Println("Interupt handler started")
		var lastTime time.Time

		for range interupts {
			if inShell.IsSet() {
				controller.SendCommand(currentShell, string(0x03))
				fmt.Println()
				return
			}

			var newTime = time.Now()
			if newTime.Sub(lastTime) <= 2*time.Second {
				printer.Println("\nExiting.")
				os.Exit(1)
			} else {
				lastTime = newTime
			}

			printer.Printf("\nUse the 'exit' command to exit or use Cntrl-C twice within 2 seconds\n")

			if !cmdRunning.IsSet() {
				prompt()
			}
		}
	}()

	dlog.Println("Starting Bluetooth Control Loop.")

	for true {
		cmdRunning.Unset()
		prompt()
		var input, _ = stdinReader.ReadString('\n')
		input = strings.TrimRight(input, "\r\n")

		var cmds = strings.Split(input, " ")

		cmdRunning.Set()

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
			controller.CancelAction(cmds[1])
		case "targets":
			targetsCmd(controller)
		case "purge-targets":
			controller.DropTargets()
		case "connect":
			connectCmd(controller, cmds)
		case "shell":
			currentShell = cmds[1]
			inShell.Set()
			shellCmd(controller, stdinReader, stdoutWriter, cmds)
			inShell.Unset()
		case "info":
			infoCmd(controller, cmds)
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

func remoteShellPrompt(id string) {
	fmt.Printf("\033[2K\r")
	printer.Printf("\033[38;5;21mbf\033[38;5;226m/\033[m%s\033[38;5;196m>\033[m ", id)
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
