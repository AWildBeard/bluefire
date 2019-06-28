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
	printer.Println("\033[4m\033[1mUse 'help' to see a list of help topics.\033[0m")
	printer.Println()

	dlog.Println("Creating bluetooth controller")
	var (
		controller = NewController()
		interupts  = make(chan os.Signal, 1)
		reader     = bufio.NewReader(os.Stdin)
	)

	dlog.Println("Starting interupt handler")
	signal.Notify(interupts, os.Interrupt)
	go func() {
		dlog.Println("Interupt handler started")
		for range interupts {
			fmt.Printf("\033[2K")
			fmt.Printf("\033[1A")
			printer.Printf("\nUse the 'exit' command to exit.\n")
			prompt()
		}
	}()

	dlog.Println("Starting Bluetooth Control Loop.")

	for true {
		prompt()
		var input, _ = reader.ReadString('\n')
		input = strings.TrimRight(input, "\r\n")

		var cmds = strings.Split(input, " ")

		dlog.Printf("Command %v\n", cmds[0])
		switch cmds[0] {
		case "?":
			fallthrough
		case "h":
			fallthrough
		case "help":
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
			var mp = controller.Targets()
			if len(mp) == 0 {
				printer.Println("Nothing here.")
				continue
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
		case "purge-targets":
			controller.DropTargets()
		case "info":
			if len(cmds) < 2 {
				printer.Println("'info' expects a target as a parameter")
			}

			if cmds[1][0] != '#' {
				printer.Println("Please select a target by its '#number'")
				continue
			}

			if target, err := controller.GetTarget(cmds[1]); err == nil {
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
	printer.Print("\033[38;5;21mbf\033[38;5;196m>\033[0m ")
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
