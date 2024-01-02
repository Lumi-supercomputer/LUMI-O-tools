package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Currently a global variable which is given a value when parsing commandline arguments
// Should probably be part of some struct...
var customRemoteName = ""

func getGenericRemoteName(projid int) string {
	if customRemoteName != "" {
		return customRemoteName
	} else {
		return fmt.Sprintf("lumi-%d", projid)

	}

}

func ValidateRemote(tmpConfigPath string, remoteName string, commandName string, fn validationFunc, printTempConfigLoc bool, skipValidation bool) (string, error) {

	if !skipValidation {
		err := fn(tmpConfigPath, remoteName)
		if err != nil {
			if printTempConfigLoc {
				fmt.Printf(configSavedmsg, commandName, tmpConfigPath, remoteName)
			}
			return fmt.Sprintf(failedRemoteValidationMsg, commandName, remoteName), err
		}
	}

	return "", nil
}

func constructDeleteList(a string) []string {
	reg, _ := regexp.Compile(`\s+`)
	return strings.Split(reg.ReplaceAllString(a, ""), ",")
}

func deleteConfigSection(programArgs Settings) error {

	sectionsToDelete := constructDeleteList(programArgs.deleteList)
	fmt.Printf("Trying to delete the following sections: %s\n", strings.Join(sectionsToDelete, " "))
	fmt.Printf("Do you want to continue (yes/no)\n")
	var response string
	var err error
	if !programArgs.nonInteractive {
		for {
			_, err := fmt.Scanf("%s", &response)
			if err != nil {
				PrintErr(err, "Unknown error when reading input")
				return err
			}
			if response == "no" {
				fmt.Printf("User respondend with no, will not continue\n")
				os.Exit(0)
			} else if response == "yes" {
				fmt.Printf("User responded with yes, continuing\n")
				break
			} else {
				fmt.Printf("Enter either yes or no\n")
			}
		}
	} else {
		fmt.Printf("Using --nonintercative, assuming yes")
	}
	for _, tool := range tools {
		if !tool.isEnabled {
			if programArgs.debug {
				fmt.Printf("Ignoring configuration for %s\n", tool.name)
				continue
			}
		} else {
			currentu, _ := user.Current()
			config := strings.Replace(tool.configPath, "~", currentu.HomeDir, -1)
			err = deleteIniSectionsFromFile(config, sectionsToDelete)
			// Extra logic for deleting configuration for aws
			if tool.name == "aws" {

				var toDel []string
				for _, x := range sectionsToDelete {
					toDel = append(toDel, strings.Join([]string{"services", x}, " "))
				}

				deleteAwsEntry(getAwsConfigFilePath(config), toDel)
			}
			if err != nil {
				PrintErr(err, "Failed while trying to delete ")
				return err
			}
		}
	}
	return nil
}

func setupArgs(settings *Settings) {
	flag.IntVar(&settings.chunksize, "chunksize", 15, `s3cmd chunk size, 5-5000, Files larger than SIZE, in MB, are automatically uploaded multithread-multipart (default: 15)`)
	flag.BoolVar(&settings.debug, "debug", false, "Keep temporary configs for debugging")
	flag.StringVar(&settings.skipValidation, "skip-validation", "", `Comma separated list of tools to skip validation for. WARNING: Might lead to a broken config`)
	flag.StringVar(&settings.keepDefault, "keep-default", "", "Comma separated list of tools to not switch defaults for. Valid values: all,s3cmd,aws")
	flag.IntVar(&settings.projectId, "project-number", 0, "Define LUMI-project to be used")
	flag.StringVar(&settings.rcloneConfig, "rclone-config", systemDefaultRcloneConfig, "Path to rclone config")
	flag.StringVar(&settings.s3cmdConfig, "s3cmd-config", systemDefaultS3cmdConfig, "Path to s3cmd config")
	flag.StringVar(&settings.awsCredentials, "aws-config", systemDefaultAwsConfig, "Path to aws credentials file. Default endpoint configuration will be added ")
	flag.StringVar(&settings.configuredTools, "configure-only", "", "Comma separated list of tools to configure for. Default is rclone,s3cmd")
	flag.BoolVar(&settings.nonInteractive, "noninteractive", false, "Read access and secret keys from environment: LUMIO_S3_ACCESS,LUMIO_S3_SECRET")
	flag.StringVar(&customRemoteName, "remote-name", "", "Custom name for the endpoints, rclone public remote name will include a -public suffix")
	flag.StringVar(&settings.deleteList, "delete", "", "Comma separated list of endpoints to delete")
	flag.StringVar(&settings.url, "url", systemDefaultS3Url, "Url for the s3 object storage")
}

func parseCommandlineArguments(settings *Settings) error {
	setupArgs(settings)
	SetCustomHelp()
	flag.Parse()
	if settings.chunksize < 5 || settings.chunksize > 5000 {
		return errors.New(fmt.Sprintf("--chunksize, Invalid Chunk size %d must be between 5 and 5000", settings.chunksize))
	}
	reg, _ := regexp.Compile(`\s+`)
	enabledTools := strings.Split(strings.ToLower(reg.ReplaceAllString(settings.configuredTools, "")), ",")
	disabledValidation := strings.Split(strings.ToLower(reg.ReplaceAllString(settings.skipValidation, "")), ",")
	keepDefaults := strings.Split(strings.ToLower(reg.ReplaceAllString(settings.keepDefault, "")), ",")
	noValidation := stringInSlice("all", disabledValidation)
	availableTools := [len(tools)]string{}
	toolMap := make(map[string]*ToolSettings)
	allEnabled := stringInSlice("all", enabledTools)
	allKeep := stringInSlice("all", keepDefaults)

	for i := range tools {
		availableTools[i] = tools[i].name
		toolMap[tools[i].name] = &tools[i]
		if settings.configuredTools != "" {
			tools[i].isEnabled = false
		}
		if allKeep {
			tools[i].noReplace = true
		}
		if allEnabled {
			tools[i].isEnabled = true
		}
		if noValidation {
			tools[i].validationDisabled = true
		}
		_, err := exec.LookPath(tools[i].name)
		if err != nil {
			tools[i].isPresent = false
		} else {
			tools[i].isPresent = false
		}

	}
	for _, et := range keepDefaults {
		if et == "rclone" {
			return errors.New("Specifying rclone for --keep-default does not make sense as rclone does not have a default remote")
		}
		if et == "all" || et == "" {
			continue
		}
		if !stringInSlice(et, availableTools[:]) {
			return errors.New(fmt.Sprintf("Unknow option %s for --keep-default flag. Valid options are: all s3cmd and aws", et))
		} else {
			toolMap[et].noReplace = true
		}
	}
	for _, et := range enabledTools {
		if et == "all" || et == "" {
			continue
		}
		if !stringInSlice(et, availableTools[:]) {
			return errors.New(fmt.Sprintf("Unknown option %s for --configure-only flag. Valid options are: all %s", et, strings.Join(availableTools[:], " ")))
		} else {
			toolMap[et].isEnabled = true
		}
	}
	for _, v := range disabledValidation {
		if v == "all" || v == "" {
			continue
		}

		if !stringInSlice(v, availableTools[:]) {
			return errors.New(fmt.Sprintf("Unknown option %s for --skip-validation flag. Valid options are: all %s", v, strings.Join(availableTools[:], " ")))

		} else {
			toolMap[v].validationDisabled = true
		}

	}

	toolMap["s3cmd"].configPath = settings.s3cmdConfig
	toolMap["rclone"].configPath = settings.rcloneConfig
	if toolMap["s3cmd"].noReplace && settings.s3cmdConfig != systemDefaultS3cmdConfig {
		fmt.Printf("WARNING: Using --keep-default s3cmd together with --s3cmd-config has no effect\n")
	}

	return nil
}

// We don't actually need to validate the projectid
// But keep it here to force the user to check what project they are generating
// access for and to see what project an endpoint was configured for without going to the webpage.
func validateProjId(id int) error {

	_, skipProjectIdValidation := os.LookupEnv("LUMIO_SKIP_PROJID")
	if skipProjectIdValidation {
		return nil
	}
	if id < 462000000 || id > 466000000 {
		invalidInputMsg := fmt.Sprintf("Invalid Lumi project number provided ( %d ), valid project numbers start with either 462 or 465 and contain 9 digits e.g 465000001", id)
		return errors.New(invalidInputMsg)
	}
	return nil
}

func getUserInput(a *AuthInfo, argProjId int) error {
	if argProjId == 0 {
		fmt.Print("Lumi project number\n")
		var inputVal string
		var err error
		i, _ := fmt.Scanf("%s", &inputVal)
		a.projectId, err = strconv.Atoi(inputVal)
		if err != nil || i == 0 {
			return errors.New("Failed to read Lumi project number, make sure there are only numbers in the input")
		}
	} else {
		a.projectId = argProjId
	}
	// Valid projects should either start with 462 or 465
	err := validateProjId(a.projectId)
	if err != nil {
		return err
	}

	fmt.Print("Access key\n")
	bytepw, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return err
	}
	a.s3AccessKey = string(bytepw)
	fmt.Printf("Secret key\n")

	bytepw, err = term.ReadPassword(syscall.Stdin)
	if err != nil {
		return err
	}
	a.s3SecretKey = string(bytepw)
	a.s3AccessKey = strings.TrimSpace(a.s3AccessKey)
	a.s3SecretKey = strings.TrimSpace(a.s3SecretKey)
	return nil
}

func getNonInteractiveInput(a *AuthInfo, argProjId int) error {
	projectIdEnvVal, projectIdEnvValIsPresent := os.LookupEnv("LUMIO_PROJECTID")
	var err error
	if argProjId != 0 {
		a.projectId = argProjId
	} else if projectIdEnvValIsPresent {
		a.projectId, err = strconv.Atoi(projectIdEnvVal)
		if err != nil {
			return errors.New("Value for LUMIO_PROJECTID needs to be a number")

		}
	} else {
		err := errors.New("--noninteractive flag used but, neither --project-number flag nor LUMIO_PROJECTID environment variable used")
		return err
	}
	err = validateProjId(a.projectId)
	if err != nil {
		return err
	}
	s3AccessKeyEnvVal, s3AccessKeyIsPresent := os.LookupEnv("LUMIO_S3_ACCESS")
	s3SecretKeyEnvVal, s3SecretKeyIsPresent := os.LookupEnv("LUMIO_S3_SECRET")

	if s3AccessKeyIsPresent && s3SecretKeyIsPresent {
		a.s3AccessKey = s3AccessKeyEnvVal
		a.s3SecretKey = s3SecretKeyEnvVal

	} else {
		err := errors.New("Both LUMIO_S3_ACCESS and LUMIO_S3_SECRET need to be set when running in noninteractive mode ")
		return err
	}

	return nil
}
