package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

func main() {

	var programArgs Settings
	var authInfo AuthInfo
	var extraInfo string

	syscall.Umask(0)

	err := parseCommandlineArguments(&programArgs)
	if err != nil {
		PrintErr(err, "Invalid input for some commandline arguments")
		os.Exit(1)
	}
	authInfo.url = programArgs.url

	if programArgs.deleteList != "" {
		err = deleteConfigSection(programArgs)
		if err != nil {
			PrintErr(err, "Failed while trying to delete endpoints")
			os.Exit(1)
		} else {

			os.Exit(0)
		}
	}

	if programArgs.nonInteractive {
		err = getNonInteractiveInput(&authInfo, programArgs.projectId)
		if err != nil {
			PrintErr(err, "Failed to start program noninteractively")
			os.Exit(1)
		}

	} else {
		fmt.Printf("%s\n", authInstructions)
		fmt.Print("\n=========== PROMPTING USER INPUT ===========\n")
		err = getUserInput(&authInfo, programArgs.projectId)
		if err != nil {
			PrintErr(err, "Invalid user input")
			os.Exit(1)
		}
	}
	authInfo.chunksize = programArgs.chunksize
	tmpDir := createTmpDir("")

	for _, tool := range tools {
		if !tool.isEnabled {
			if programArgs.debug {
				fmt.Printf("Skipping configuration for %s\n", tool.name)
			}
		} else {
			fmt.Printf("\n=========== CONFIGURING %s ===========\n", strings.ToUpper(tool.name))
			if tool.validationDisabled {
				fmt.Printf("%s\n\n", skipValidationWarning)
			}
			extraInfo, err = tool.addRemote(authInfo, tmpDir, programArgs.debug, tool)
			if err != nil {
				if !tool.isPresent {
					fmt.Printf("WARNING: %s command missing (if %s is a shell alias this script will not find it)\n", tool.name, tool.name)
				}
				PrintErr(err, extraInfo)
			}
		}
	}
	if !programArgs.debug {
		os.RemoveAll(tmpDir)
	}
}
