package main

var (
	commandDescriptions = map[string]string{
		"targets":       "targets list the targets that have been discovered by scan.",
		"ps":            "ps lists the running actions. Actions can be killed using the\nkill command.",
		"info":          "info returns the advertisement info for a ble device. info takes\na #number from the target command as an argument.",
		"kill":          "kill stops a background action. It expects an argument from the\noutput of ps to run",
		"scan":          "scan starts the scan action in the background. This populates\nthe targets command.",
		"purge-targets": "purge-targets removes all targets from the targets command.",
		"clear":         "clear clears the terminal screen.",
		"exit":          "exit exits BlueFire.",
		"help":          "help prints the default help page. It can also take the name of\na command as an argument to return more information about that command.",
	}
)

func validCommands() []string {
	return []string{"\033[1mtargets\033[m",
		"\033[1mps\033[m",
		"\033[1minfo \033[0;4mtarget\033[0m"}
}

func validActions() []string {
	return []string{"\033[1mkill \033[0;4maction\033[0m",
		"\033[1mscan\033[m", "\033[1mpurge-targets\033[m",
		"\033[1mshell\033[m \033[4mtarget\033[m"}
}

func validUtilities() []string {
	return []string{"\033[1mclear\033[m",
		"\033[1mexit\033[m",
		"\033[1mhelp\033[m [\033[4mcommand\033[m]"}
}

func commandInfo(cmd string) string {
	return commandDescriptions[cmd]
}
