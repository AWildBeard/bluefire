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

	dlog.Println("Creating bluetooth controller")
	var (
		controller = NewController()
	)

	dlog.Println("Starting Bluetooth Control Thread.")

	var reader = bufio.NewReader(os.Stdin)
	for true {
		printer.Print("bf> ")
		var input, _ = reader.ReadString('\n')
		input = strings.TrimRight(input, "\r\n")

		var cmds = strings.Split(input, " ")

		dlog.Printf("Command %v\n", cmds[0])
		switch cmds[0] {
		case "help":
			fivePrint(ValidActions())
		case "scan":
			if err := controller.ScanNow(); err != nil {
				printer.Printf("%v\n", err)
			}
		case "ps":
			fivePrint(controller.RunningActions())
		case "cancel":
			controller.CancelAction(cmds[1])
		case "targets":
			var mp = controller.Targets()
			if len(mp) == 0 {
				printer.Println("Nothing here.")
				continue
			}

			for key, value := range mp {
				printer.Printf("%v  :  %v\n", key, value)
			}
		case "purge-targets":
			controller.DropTargets()
		default:
			printer.Println("Please enter a valid command")
		}
	}

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
