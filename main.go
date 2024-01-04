package main

import (
	"fmt"
	"lumioconf/toolConfig"

	"lumioconf/util"
	"os"
	"strings"
	"syscall"
)

func main() {
	var toolMap = map[string]*toolConfig.ToolSettings{
		"rclone": &toolConfig.RcloneSettings,
		"s3cdm":  &toolConfig.S3cmdSettings,
		"aws":    &toolConfig.AwsSettings}

	var programArgs toolConfig.Settings
	var authInfo toolConfig.AuthInfo
	var extraInfo string

	syscall.Umask(0)

	err := toolConfig.ParseCommandlineArguments(&programArgs, toolMap)
	if err != nil {
		util.PrintErr(err, "Invalid input for some commandline arguments")
		os.Exit(1)
	}
	authInfo.Url = programArgs.Url

	if programArgs.DeleteList != "" {
		err = toolConfig.DeleteConfigSection(programArgs, toolMap)
		if err != nil {
			util.PrintErr(err, "Failed while trying to delete endpoints")
			os.Exit(1)
		} else {

			os.Exit(0)
		}
	}

	if programArgs.NonInteractive {
		err = toolConfig.GetNonInteractiveInput(&authInfo, programArgs.ProjectId)
		if err != nil {
			util.PrintErr(err, "Failed to start program noninteractively")
			os.Exit(1)
		}

	} else {
		fmt.Printf("%s\n", toolConfig.AuthInstructions)
		fmt.Print("\n=========== PROMPTING USER INPUT ===========\n")
		err = toolConfig.GetUserInput(&authInfo, programArgs.ProjectId)
		if err != nil {
			util.PrintErr(err, "Invalid user input")
			os.Exit(1)
		}
	}
	authInfo.Chunksize = programArgs.Chunksize
	tmpDir := util.CreateTmpDir("")

	for _, tool := range toolMap {
		if !tool.IsEnabled {
			if util.GlobalDebugFlag {
				fmt.Printf("Skipping configuration for %s\n", tool.Name)
			}
		} else {
			fmt.Printf("\n=========== CONFIGURING %s ===========\n", strings.ToUpper(tool.Name))
			if tool.ValidationDisabled {
				fmt.Printf("%s\n\n", toolConfig.SkipValidationWarning)
			}
			extraInfo, err = tool.AddRemote(authInfo, tmpDir, *tool)
			if err != nil {
				if !tool.IsPresent {
					fmt.Printf("WARNING: %s command missing (if %s is a shell alias this script will not find it)\n", tool.Name, tool.Name)
				}
				util.PrintErr(err, extraInfo)
			}
		}
	}
	if !util.GlobalDebugFlag {
		os.RemoveAll(tmpDir)
	}
}
