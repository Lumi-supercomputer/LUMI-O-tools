package toolConfig

import (
	"errors"
	"flag"
	"fmt"
	"lumioconf/internal/util"
	"os"
	"os/exec"
	"os/user"
	"reflect"
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

func DeleteConfigSection(programArgs Settings, toolMap map[string]*ToolSettings) error {

	sectionsToDelete := util.RemoveWhiteSpaceAndSplit(programArgs.DeleteList)
	fmt.Printf("Trying to delete the following sections: %s\n", strings.Join(sectionsToDelete, " "))
	fmt.Printf("Do you want to continue (yes/no)\n")
	var response string
	var err error
	if !programArgs.NonInteractive {
		for {
			_, err := fmt.Scanf("%s", &response)
			if err != nil {
				util.PrintErr(err, "Unknown error when reading input")
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
		fmt.Printf("Using --nonintercative, assuming yes\n")
	}
	for _, tool := range toolMap {
		if !tool.IsEnabled {
			if util.GlobalDebugFlag {
				fmt.Printf("Ignoring configuration for %s\n", tool.Name)
				continue
			}
		} else {
			currentu, _ := user.Current()
			config := strings.Replace(tool.configPath, "~", currentu.HomeDir, 1)
			err = util.DeleteIniSectionsFromFile(config, sectionsToDelete)
			// Extra logic for deleting configuration for aws
			if tool.Name == "aws" {

				var toDel []string
				for _, x := range sectionsToDelete {
					toDel = append(toDel, strings.Join([]string{"services", x}, " "))
				}

				err = deleteAwsEntry(getAwsConfigFilePath(config), toDel)
				if err != nil {
					return err
				}
			}
			if tool.Name == "s3cmd" {
				err = deleteExtraS3cmdConfig(tool.configPath, sectionsToDelete)
				if err != nil {
					return err
				}

			}
		}
	}
	return nil
}

func ValidateRemote(tmpConfigPath string, remoteName string, commandName string, fn validationFunc, skipValidation bool) (string, error) {

	if !skipValidation {
		err := fn(tmpConfigPath, remoteName)
		if err != nil {
			if util.GlobalDebugFlag {
				fmt.Printf(configSavedmsg, commandName, tmpConfigPath, remoteName)
			}
			return fmt.Sprintf(failedRemoteValidationMsg, commandName, remoteName), err
		}
	}

	return "", nil
}

func parseKeepMapping(confArg string) (map[string]bool, error) {
	stringMap := util.RemoveWhiteSpaceAndSplit(confArg)
	mappings := make(map[string]bool)
	for _, mapping := range stringMap {
		m := strings.Split(mapping, ":")
		if len(m) != 2 {
			newErr := errors.New(fmt.Sprintf("Incorrect format for argument to --keep-default. Is %s, should be tool1:true,tool2:false", confArg))
			return nil, newErr
		} else {

			boolVal, err := strconv.ParseBool(m[1])
			if err != nil {
				return nil, errors.New(fmt.Sprintf("Error while parsing argument to --keep-default=%s. Error: %s", confArg, err.Error()))
			}
			mappings[m[0]] = boolVal
		}
	}
	return mappings, nil
}

func parseConfigPathMapping(confArg string) (map[string]string, error) {
	stringMap := util.RemoveWhiteSpaceAndSplit(confArg)
	mappings := make(map[string]string)
	for _, mapping := range stringMap {
		m := strings.Split(mapping, ":")
		if len(m) != 2 {
			newErr := errors.New(fmt.Sprintf("Incorrect format for argument to --config-path. Is %s, should be tool1:path1,tool2:path2", confArg))
			return nil, newErr
		} else {

			// We were passed a directory or something which looks like a directory
			if strings.HasSuffix(m[1], "/") || util.IsDirectory(m[1]) {
				newErr := errors.New(fmt.Sprintf("Incorrect argument to --config-path. %s:%s\n\tPath can not end with / or be an existing directory. Specify the full path to the config file", m[0], m[1]))
				return nil, newErr
			}
			mappings[m[0]] = m[1]
		}
	}

	return mappings, nil
}

func validateChunksize(settings *Settings) error {
	if settings.Chunksize < 5 || settings.Chunksize > 5000 {
		return errors.New(fmt.Sprintf("--chunksize, Invalid Chunk size %d must be between 5 and 5000", settings.Chunksize))
	}
	return nil
}

func ParseCommandlineArguments(settings *Settings, toolMap map[string]*ToolSettings) error {
	var configuredTools string
	var skipValidation string
	var keepDefault string
	var configPathMapping string
	flag.StringVar(&configPathMapping, "config-path", "", "Comma separated list of config paths for the tools. E.g rclone:/path/to/configFile,s3cmd:/path/to/config2File")
	flag.StringVar(&skipValidation, "skip-validation", "", `Comma separated list of tools to skip validation for. WARNING: Might lead to a broken config`)
	flag.StringVar(&keepDefault, "keep-default", "", "Comma separated list of tools to not switch defaults for. Default value: s3cmd:true,aws:false")
	flag.StringVar(&configuredTools, "configure-only", "", "Comma separated list of tools to create configurations for. Default is rclone and s3cmd")
	flag.IntVar(&settings.Chunksize, "chunksize", 15, `s3cmd and aws cli chunk size, 5-5000, Files larger than SIZE, in MB, are automatically uploaded multithread-multipart (default: 15)`)
	flag.BoolVar(&util.GlobalDebugFlag, "debug", false, "Keep temporary configs for debugging and display additional output")
	flag.IntVar(&settings.ProjectId, "project-number", 0, "Define LUMI-project to be used")
	flag.BoolVar(&settings.NonInteractive, "noninteractive", false, "Read access and secret keys from environment: LUMIO_S3_ACCESS,LUMIO_S3_SECRET")
	flag.StringVar(&customRemoteName, "remote-name", "", "Custom name for the endpoints, rclone public remote name will include a -public suffix")
	flag.StringVar(&settings.DeleteList, "delete", "", "Comma separated list of endpoints to delete")
	flag.StringVar(&settings.Url, "url", systemDefaultS3Url, "Url for the s3 object storage")
	flag.BoolVar(&settings.ShowVersion, "version", false, "Show version information and exit")
	util.SetCustomHelp()
	flag.Parse()
	// Exit early if --version was given
	if settings.ShowVersion {
		return nil
	}

	availableTools := make([]string, len(toolMap))
	i := 0
	for k := range toolMap {
		availableTools[i] = k
		i++
	}

	checkIfPresent(toolMap)

	err := setEnabledTools(configuredTools, availableTools, toolMap)
	if err != nil {
		return err
	}
	err = setConfigPaths(configPathMapping, availableTools, toolMap)
	if err != nil {
		return err
	}
	if strings.Contains(keepDefault, "rclone") {
		return errors.New("Specifying rclone for --keep-default does not make sense as rclone does not have a default remote\n")
	}

	err = setKeepDefaultToggle(keepDefault, util.RemoveStringFromSlice(availableTools, "rclone"), toolMap)
	if err != nil {
		return err
	}
	err = disableValidationForSelectedTools(skipValidation, availableTools, toolMap)
	if err != nil {
		return err
	}
	if toolMap["s3cmd"].NoReplace && toolMap["s3cmd"].configPath != systemDefaultConfigPaths["s3cmd"] {
		fmt.Printf("WARNING: Using --keep-default s3cmd together with --s3cmd-config has no effect\n")
	}

	// Chuncksize option is not used for rlcone so don't verify unless needed.
	if toolMap["s3cmd"].IsEnabled || toolMap["aws"].IsEnabled {
		validateChunksize(settings)
	}

	return nil
}

func checkIfPresent(toolMap map[string]*ToolSettings) {
	for k := range toolMap {
		_, err := exec.LookPath(k)
		if err != nil {
			toolMap[k].IsPresent = false
		} else {
			toolMap[k].IsPresent = true
		}

	}
}

// A non-empty input means we have to make sure
// No tool will default to being active
func setEnabledTools(toEnable string, available []string, toolMap map[string]*ToolSettings) error {
	if toEnable != "" {
		for _, v := range toolMap {
			v.IsEnabled = false
		}
	}
	return genericSetter(toEnable, available, "IsEnabled", "--configure-only", toolMap)
}

func setKeepDefaultToggle(toolNamesToKeepDefaultsS string, available []string, toolMap map[string]*ToolSettings) error {
	if toolNamesToKeepDefaultsS == "" {
		return nil
	}
	keepBools, err := parseKeepMapping(toolNamesToKeepDefaultsS)
	if err != nil {
		return err
	}
	for k, v := range keepBools {
		if !util.StringInSlice(k, available[:]) {
			return errors.New(fmt.Sprintf("Unknown toolname %s in --keep-default", k))
		} else {
			fmt.Printf("Bool val for %s %t\n", k, v)
			toolMap[k].NoReplace = v
		}
	}

	return nil
}

func setKeepDefault(toolNamesToKeepDefaultsS string, available []string, toolMap map[string]*ToolSettings) error {
	return genericSetter(toolNamesToKeepDefaultsS, available, "NoReplace", "--keep-default", toolMap)
}

func disableValidationForSelectedTools(toolNamesToDisableS string, available []string, toolMap map[string]*ToolSettings) error {
	return genericSetter(toolNamesToDisableS, available, "ValidationDisabled", "--skip-validation", toolMap)
}

func setConfigPaths(pathM string, available []string, toolMap map[string]*ToolSettings) error {
	if pathM == "" {
		return nil
	}
	configPaths, err := parseConfigPathMapping(pathM)
	if err != nil {
		return err
	}

	for k, v := range configPaths {
		if !util.StringInSlice(k, available[:]) {
			return errors.New(fmt.Sprintf("Unknown toolname %s in --config-path.", k))
		} else {
			toolMap[k].configPath = v
		}

	}
	return nil
}

func genericSetter(argString string, toolNames []string, propertyName string, optionName string, toolMap map[string]*ToolSettings) error {
	if argString == "" {
		return nil
	}
	l := util.RemoveWhiteSpaceAndSplit(argString)

	for _, et := range l {
		if et == "all" {
			for _, t := range toolNames {
				reflect.ValueOf(toolMap[t]).Elem().FieldByName(propertyName).SetBool(true)
			}
			return nil
		}
		if !util.StringInSlice(et, toolNames) {
			return errors.New(fmt.Sprintf("Unknow option %s for %s flag. Valid options are: all %s", et, optionName, strings.Join(toolNames[:], " ")))

		} else {
			reflect.ValueOf(toolMap[et]).Elem().FieldByName(propertyName).SetBool(true)

		}
	}
	return nil
}

// We don't actually need to validate the projectid
// But keep it here to force the user to check what project they are generating
// access for and to see what project an endpoint was configured for without going to the webpage.
// Additionally we can also print the correct public url for rclone objects.
// This is just a sanity check as we cannot check if the whole number is correct
// Just the first digits and the number of digits
// 462 465 442
func validateProjId(id int) error {

	var idStartAsString string
	_, skipProjectIdValidation := os.LookupEnv("LUMIO_SKIP_PROJID_CHECK")
	if skipProjectIdValidation {
		return nil
	}
	projIdLen := 9 // 465000001
	idAsString := fmt.Sprintf("%d", id)
	if len(idAsString) > 2 {
		idStartAsString = idAsString[0:3]
	} else {
		idStartAsString = idAsString
	}
	if !(idStartAsString == "462" || idStartAsString == "465" || idStartAsString == "442") || projIdLen != len(idAsString) {
		util.PrintVerb(fmt.Sprintf("Projectid starts with %s and is %d digits\n", idStartAsString, len(idAsString)))
		invalidInputMsg := fmt.Sprintf("Invalid Lumi project number provided ( %d ), valid project numbers start with either 462 or 465 and contain 9 digits e.g 465000001", id)
		return errors.New(invalidInputMsg)
	}
	return nil
}

func GetUserInput(a *AuthInfo, argProjId int) error {
	if argProjId == 0 {
		fmt.Print("Lumi project number\n")
		var inputVal string
		var err error
		i, _ := fmt.Scanf("%s", &inputVal)
		a.ProjectId, err = strconv.Atoi(inputVal)
		if err != nil || i == 0 {
			return errors.New("Failed to read Lumi project number, make sure there are only numbers in the input")
		}
	} else {
		a.ProjectId = argProjId
	}
	// Valid projects should either start with 462 or 465
	err := validateProjId(a.ProjectId)
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

func GetNonInteractiveInput(a *AuthInfo, argProjId int) error {
	projectIdEnvVal, projectIdEnvValIsPresent := os.LookupEnv("LUMIO_PROJECTID")
	var err error
	if argProjId != 0 {
		a.ProjectId = argProjId
	} else if projectIdEnvValIsPresent {
		a.ProjectId, err = strconv.Atoi(projectIdEnvVal)
		if err != nil {
			return errors.New("Value for LUMIO_PROJECTID needs to be a number")

		}
	} else {
		err := errors.New("--noninteractive flag used but, neither --project-number flag nor LUMIO_PROJECTID environment variable used")
		return err
	}
	err = validateProjId(a.ProjectId)
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
