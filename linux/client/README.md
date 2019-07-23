### This folder contains the third component (the attacking client) to BlueFire

#### Building the BlueFire client
First, decide which architecture you're going to be building for. Currently the makefile has options for arm and amd64. The following docs are going to assume arm
* For a release build: `make release arm-linux`
* For a debug build: `make arm-linux`

It's worth noting that the `-debug` flag works the same for both a release binary
and a debug binary. The main difference between the two is that the debugging symbols for a panic have been stripped out of the release-mode binary.

#### Using the BlueFire client
Simply run the `bluefire` binary, and it will be ready to go! See the command
specification below for details on available commands

#### Commands
> help
* This command will display a useful list of help information that will allow the user to use the client. All valid commands will be displayed. No commands will be hidden from the user. This command also has an extended form that allows passing another command to it as an argument that will cause the help command to spit out helpful information only for that command


> targets
* This command will display all targets that have currently been identified by the client. These targets are pre-checked for specific identifiers that allow BlueFire to positively identify only targets that are actually running BlueFire


> ps
* This command will list all of the background processes that are running. This will include 'scan' and any connections that are actively being maintained by the firmware device.


> info target
* This command allows the user to find out more useful information about a target, such as it's RSSI, if it is serving any other bluetooth services, etc.


> kill action
* This command allows the user to tell the program, and the bluetooth firmware chip, to end a currently running task. For example, this command can be used to cancel the scanning process


> scan
* This command will start the scanning process. Results will automatically be added to the internal targets list that can be viewed with the 'targets' command.


> purge-targets
* This command will delete all the targets in the current target list. If scanning is running, this will cause scanning to stop and restart which will repopulate the targets list with any targets that are nearby. This is usefull if the attacker is mobile. kill the scanning process with the kill command, and then purge-targets to just delete the targets list.


> shell target
* This command allows the user to access the direct command line environment of the target they have selected. To exit the shell environment, use the character combo Ctrl-b ; q to exit the shell and return to BlueFire


> clear 
* This command clears the BlueFire terminal screen just as it would for bash on linux or as cls would on Windows.


> exit
* This command will exit BlueFire