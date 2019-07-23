package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

	for true {
		prompt()
		var input, _ = stdinReader.ReadString('\n')
		input = strings.TrimRight(input, "\r\n")

		var cmds = strings.Split(input, " ")

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
