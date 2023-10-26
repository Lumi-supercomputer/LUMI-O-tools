package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
	"gopkg.in/ini.v1"
)

type Settings struct {
	chunksize              int
	debug                  bool
	projectId              int
	skipValidation         string
	keepDefaultS3cmdConfig bool
	rcloneConfig           string
	s3cmdConfig            string
	configuredTools        string
	nonInteractive         bool
}
type AuthInfo struct {
	s3AccessKey string
	s3SecretKey string
	projectId   int
	chunksize   int
}

func getGenericRemoteName(projid int) string {
	if customRemoteName != "" {
		return customRemoteName
	} else {
		return fmt.Sprintf("lumi-%d", projid)

	}

}

func getPublicRcloneRemoteName(projid int) string {
	if customRemoteName != "" {
		return fmt.Sprintf("%s-public", customRemoteName)
	} else {
		return fmt.Sprintf("lumi-%d-public", projid)
	}
}
func getPrivateRcloneRemoteName(projid int) string {
	if customRemoteName != "" {
		return customRemoteName
	} else {
		return fmt.Sprintf("lumi-%d-private", projid)
	}
}

const skipValidationWarning = `WARNING: The --skip-validation flag was used, configurations will not be validated and could potentially be saved in an invalid state if user input is incorrect`

const failedRemoteValidationMsg = `Failed to validate new %s endpoint %s
No new endpoint was added
Double check that the correct details were enter
Run with --debug to keep the generated temporary configuration
The error was:`

const configSavedmsg = `Generated %s config has been saved to %s
IMPORTANT: When troubleshooting, DO NOT share the whole file
ONLY the info related to the specific failed endpoint %s

`
const passedRcloneRemoteValdidationMessage = `rclone remote %s: now provides an S3 based connection to Lumi-O storage area of project_%d

rclone remote %s: now provides an S3 based connection to Lumi-O storage area of project_%d
	Data pushed here is publicly available using the URL: https://%d.lumidata.eu/<bucket_name>/<object>"
`

const passedS3cmdRemoteValidationMessage = `Created s3cmd config for project_%d
	Other existing configurations can be accessed by adding the -c flag
	s3cdm -c ~/.s3cfg-lumio-<project_number> COMMAND ARGS
`
const noUpdates3cfgMessage = `Default s3cmd config was not chaged, current default is %s in file %s
Either set S3CMD_CONFIG
Or use the -c flag on the commandline to use the generated config

`
const authInstructions = `Please login to  https://auth.lumidata.eu/
In the web interface, choose first the project you wish to use.
Next generate a new key or use existing valid key
Open the Key details view and based on that give following information`

var programName = filepath.Base(os.Args[0])
var customRemoteName = ""

func setupArgs(settings *Settings) {
	flag.IntVar(&settings.chunksize, "chunksize", 15, `s3cmd chunk size, 5-5000, Files larger than SIZE, in MB, are automatically uploaded multithread-multipart (default: 15)`)
	flag.BoolVar(&settings.debug, "debug", false, "Keep temporary configs for debugging")
	flag.StringVar(&settings.skipValidation, "skip-validation", "", `Comma separated list of tools to skip validation for. WARNING: Might lead to a broken config`)
	flag.BoolVar(&settings.keepDefaultS3cmdConfig, "keep-default-s3cmd-config", false, "Don't set the new configuration as default for s3cmd")
	flag.IntVar(&settings.projectId, "project-number", 0, "Define LUMI-project to be used")
	flag.StringVar(&settings.rcloneConfig, "rclone-config", systemDefaultRcloneConfig, "Path to rclone config")
	flag.StringVar(&settings.s3cmdConfig, "s3cmd-config", systemDefaultS3cmdConfig, "Path to s3cmd config")
	flag.StringVar(&settings.configuredTools, "configure-only", "", "Comma separated list of tools to configure for. Default is rclone,s3cmd")
	flag.BoolVar(&settings.nonInteractive, "noninteractive", false, "Read access and secret keys from environment: LUMIO_S3_ACCESS,LUMIO_S3_SECRET")
	flag.StringVar(&customRemoteName, "remote-name", "", "Custom name for the endpoints, rclone public remote name will include a -public suffix")
}

func parseCommandlineArguments(settings *Settings) error {
	setupArgs(settings)
	SetCustomHelp()
	flag.Parse()
	if settings.keepDefaultS3cmdConfig && settings.s3cmdConfig != systemDefaultS3cmdConfig {
		fmt.Printf("WARNING: Using --keep-default-s3cmd-config together with --s3cmd-config has no effect\n")
	}
	if settings.chunksize < 5 || settings.chunksize > 5000 {
		return errors.New(fmt.Sprintf("--chunksize, Invalid Chunk size %d must be between 5 and 5000", settings.chunksize))
	}
	reg, _ := regexp.Compile(`\s+`)
	enabledTools := strings.Split(strings.ToLower(reg.ReplaceAllString(settings.configuredTools, "")), ",")
	disabledValidation := strings.Split(strings.ToLower(reg.ReplaceAllString(settings.skipValidation, "")), ",")
	noValidation := stringInSlice("all", disabledValidation)
	availableTools := [len(tools)]string{}
	toolMap := make(map[string]*ToolSettings)
	allEnabled := stringInSlice("all", enabledTools)

	for i := range tools {
		availableTools[i] = tools[i].name
		toolMap[tools[i].name] = &tools[i]
		if settings.configuredTools != "" {
			tools[i].isEnabled = false
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
	toolMap["s3cmd"].noReplace = settings.keepDefaultS3cmdConfig
	toolMap["rclone"].configPath = settings.rcloneConfig

	return nil
}

func validateProjId(id int) error {

	if id < 462000000 || id > 466000000 {
		invalidInputMsg := fmt.Sprintf("Invalid Lumi project number provided ( %d ), valid project numbers start with either 462 or 465 and contain 9 digits e.g 465000001", id)
		return errors.New(invalidInputMsg)
	}
	return nil
}

func getUserInput(a *AuthInfo, argProjId int) error {
	if argProjId == 0 {
		fmt.Print("Lumi project number\n")
		i, err := fmt.Scanf("%d", &a.projectId)
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

func ValidateRcloneRemote(rcloneConfigFilePath string, remoteName string) error {
	os.Setenv("RCLONE_CONFIG", rcloneConfigFilePath)
	command_args := fmt.Sprintf("%s:", remoteName)
	return checkCommand("rclone", "lsd",
		"--contimeout", "2s",
		"--timeout", "2s",
		"--low-level-retries", "1",
		"--retries", "1",
		command_args)
}

func ValidateS3cmdRemote(s3cmdConfigFilePath string, remoteName string) error {
	return checkCommand("s3cmd", "-c", s3cmdConfigFilePath, "ls", "s3:")
}

func getS3cmdSetting(a AuthInfo) map[string]map[string]string {
	s3cmdSettings := make(map[string]map[string]string)
	s3cmdSettings[getGenericRemoteName(a.projectId)] = map[string]string{"access_key": a.s3AccessKey,
		"secret_key":           a.s3SecretKey,
		"host_base":            "https://lumidata.eu",
		"host_bucket":          "https://lumidata.eu",
		"human_readable_sizes": "True",
		"enable_multipart":     "True",
		"signature_v2":         "True",
		"use_https":            "True",
		"chunk_size":           fmt.Sprintf("%d", a.chunksize)}
	return s3cmdSettings

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

func adds3cmdRemote(s3auth AuthInfo, tmpDir string, printTempConfigInfo bool, s3cmdSettings ToolSettings) (string, error) {

	currentu, _ := user.Current()
	s3cmdBaseConfigPath := fmt.Sprintf("%s", strings.Replace(s3cmdSettings.configPath, "~", currentu.HomeDir, -1))
	nonDefaultConfigPathSet := s3cmdSettings.configPath != systemDefaultS3cmdConfig
	s3cmdConfigPath := s3cmdBaseConfigPath
	tmps3cmdConfig := fmt.Sprintf("%s/temp_s3cmd.config", tmpDir)
	remoteName := getGenericRemoteName(s3auth.projectId)
	updateConfig(getS3cmdSetting(s3auth), s3cmdConfigPath, tmps3cmdConfig, s3cmdSettings.carefullUpdate, s3cmdSettings.singleSection)
	info, err := ValidateRemote(tmps3cmdConfig, remoteName, "s3cmd", ValidateS3cmdRemote, printTempConfigInfo, s3cmdSettings.validationDisabled)
	if err != nil {
		return info, err
	}

	if _, err := os.Stat(s3cmdBaseConfigPath); errors.Is(err, os.ErrNotExist) {
		if s3cmdSettings.noReplace {
			fmt.Printf("WARNING: --keep-default-s3cmd-config used, but %s does not exists\n", s3cmdBaseConfigPath)
		}
	}

	if !nonDefaultConfigPathSet && s3cmdSettings.noReplace {
		s3cmdConfigPath = fmt.Sprintf("%s-%s", s3cmdBaseConfigPath, getGenericRemoteName(s3auth.projectId))
	}

	inf, err := commitTempConfigFile(tmps3cmdConfig, s3cmdConfigPath)
	if err != nil {

		return fmt.Sprintf("While updating configuration, %s", inf), err
	}
	if !s3cmdSettings.noReplace && !nonDefaultConfigPathSet {
		fmt.Printf("Updated s3cmd config %s\n\n", s3cmdConfigPath)
	} else {
		if s3cmdSettings.noReplace && !nonDefaultConfigPathSet {
			fmt.Printf("Saved generated config to %s\n", s3cmdConfigPath)
			cfg, err := ini.Load(s3cmdBaseConfigPath)
			if err == nil {
				fmt.Printf(noUpdates3cfgMessage, cfg.Sections()[1].Name(), s3cmdBaseConfigPath)
			} else {
				fmt.Printf("No default configuration exists, use S3CMD_CONFIG or the -c flag to use the generated config\n")
			}
		}
	}

	return "", nil

}

func getRcloneSetting(a AuthInfo) map[string]map[string]string {
	rcloneSettings := make(map[string]map[string]string)
	privateRemoteName := getPrivateRcloneRemoteName(a.projectId)
	publicRemoteName := getPublicRcloneRemoteName(a.projectId)
	sharedRemoteSettings := map[string]string{
		"type":              "s3",
		"provider":          "Ceph",
		"env_auth":          "false",
		"access_key_id":     a.s3AccessKey,
		"secret_access_key": a.s3SecretKey,
		"endpoint":          "https://lumidata.eu"}
	rcloneSettings[privateRemoteName] = MergeMaps(map[string]string{"acl": "private"}, sharedRemoteSettings)
	rcloneSettings[publicRemoteName] = MergeMaps(map[string]string{"acl": "public"}, sharedRemoteSettings)

	return rcloneSettings
}

func addRcloneRemotes(s3auth AuthInfo, tmpDir string, printTempConfigInfo bool, rcloneSettings ToolSettings) (string, error) {
	currentu, _ := user.Current()
	rcloneConfigPath := strings.Replace(rcloneSettings.configPath, "~", currentu.HomeDir, -1)
	tmpRcloneConfig := fmt.Sprintf("%s/temp_rclone.config", tmpDir)
	updateConfig(getRcloneSetting(s3auth), rcloneConfigPath, tmpRcloneConfig, rcloneSettings.carefullUpdate, rcloneSettings.singleSection)
	remoteName := getPrivateRcloneRemoteName(s3auth.projectId)
	info, err := ValidateRemote(tmpRcloneConfig, remoteName, "rclone", ValidateRcloneRemote, printTempConfigInfo, rcloneSettings.validationDisabled)
	if err != nil {
		return info, err
	}
	inf, err := commitTempConfigFile(tmpRcloneConfig, rcloneConfigPath)

	if err != nil {

		return fmt.Sprintf("While updating configuration, %s", inf), err
	}

	fmt.Printf("Updated rclone config %s\n\n", rcloneConfigPath)
	fmt.Printf(passedRcloneRemoteValdidationMessage, remoteName, s3auth.projectId, getPublicRcloneRemoteName(s3auth.projectId), s3auth.projectId, s3auth.projectId)
	return "", nil
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

	if programArgs.nonInteractive {
		err = getNonInteractiveInput(&authInfo, authInfo.projectId)
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
