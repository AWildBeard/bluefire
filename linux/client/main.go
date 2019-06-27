package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

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

}
