package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/ini.v1"
)

type Settings struct {
	chunksize              int
	debug                  bool
	projectId              int
	skipValidation         bool
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

const skipValidationWarning = `WARNING: The --skip-validation was used, configurations will not be validated and could potentially be saved in an invalid state if user input is incorrect`

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
const noUpdates3cfgMessage = `Default s3cmd config was not chaged, current default is %s
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
	flag.BoolVar(&settings.skipValidation, "skip-validation", false, `Don't verify endpoint configuration before saving WARNING: Might lead to a broken config`)
	flag.BoolVar(&settings.keepDefaultS3cmdConfig, "keep-default-s3cmd-config", false, "Don't set the new configuration as default for s3cmd")
	flag.IntVar(&settings.projectId, "project-number", -1, "Define LUMI-project to be used")
	flag.StringVar(&settings.rcloneConfig, "rclone-config", systemDefaultRcloneConfig, "Path to rclone config")
	flag.StringVar(&settings.s3cmdConfig, "s3cmd-config", systemDefaultS3cmdConfig, "Path to s3cmd config")
	flag.StringVar(&settings.configuredTools, "configure-only")
	flag.BoolVar(&settings.nonInteractive, "noninteractive", false, "Read access and secret keys from environment: LUMIO_S3_ACCESS,LUMIO_S3_SECRET")
	flag.StringVar(&customRemoteName, "remote-name", "", "Custom name for the endpoints, rclone public remote name will include a -public suffix")
}

func parseCommandlineArguments(settings *Settings) error {
	setupArgs(settings)
	SetCustomHelp()
	flag.Parse()
	if settings.chunksize < 5 || settings.chunksize > 5000 {
		return errors.New(fmt.Sprintf("--chunksize, Invalid Chunk size %d must be between 5 and 5000", settings.chunksize))
	}
	if settings.skipRcloneConfiguration && settings.skipS3cmdConfiguration {
		return errors.New("Told to skip configuration for all known tools ( rclone and s3cmd ), nothing to do, exiting")
	}
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
	if argProjId == -1 {
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
	fmt.Scanf("%s", &a.s3AccessKey)
	fmt.Printf("Secret key\n")
	fmt.Scanf("%s", &a.s3SecretKey)
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

func adds3cmdRemote(s3auth AuthInfo, s3cmdConfigPathNotExpanded string, tmpDir string, skipValidation bool, printTempConfigInfo bool, keepDefaultCfg bool) (string, error) {

	currentu, _ := user.Current()
	s3cmdBaseConfigPath := fmt.Sprintf("%s", strings.Replace(s3cmdConfigPathNotExpanded, "~", currentu.HomeDir, -1))
	nonDefaultConfigPathSet := s3cmdConfigPathNotExpanded != systemDefaultS3cmdConfig
	s3cmdConfigPath := s3cmdBaseConfigPath
	if !nonDefaultConfigPathSet {
		s3cmdConfigPath = fmt.Sprintf("%s-lumio-%d", s3cmdBaseConfigPath, s3auth.projectId)

	}
	tmps3cmdConfig := fmt.Sprintf("%s/temp_s3cmd.config", tmpDir)
	remoteName := getGenericRemoteName(s3auth.projectId)
	updateConfig(getS3cmdSetting(s3auth), tmps3cmdConfig, carefullUpdate)
	info, err := ValidateRemote(tmps3cmdConfig, remoteName, "s3cmd", ValidateS3cmdRemote, printTempConfigInfo, skipValidation)
	if err != nil {
		return info, err
	}

	inf, err := commitTempConfigFile(tmps3cmdConfig, s3cmdConfigPath)

	if err != nil {

		return fmt.Sprintf("While updating configuration, %s", inf), err
	}
	fmt.Printf("Updated s3cmd config %s\n\n", s3cmdConfigPath)
	if !keepDefaultCfg && !nonDefaultConfigPathSet {
		fmt.Printf("Switching default s3cmd (%s) config to %s\n", s3cmdBaseConfigPath, s3cmdConfigPath)
		os.Remove(s3cmdBaseConfigPath)
		os.Symlink(s3cmdConfigPath, s3cmdBaseConfigPath)
		fmt.Printf(passedS3cmdRemoteValidationMessage, s3auth.projectId)
	} else {
		if keepDefaultCfg && !nonDefaultConfigPathSet {
			cfg, _ := ini.Load(s3cmdBaseConfigPath)
			fmt.Printf(noUpdates3cfgMessage, cfg.Sections()[1].Name())
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

func addRcloneRemotes(s3auth AuthInfo, rcloneConfigPathNotExpanded string, tmpDir string, skipValidation bool, printTempConfigInfo bool) (string, error) {
	currentu, _ := user.Current()
	rcloneConfigPath := strings.Replace(rcloneConfigPathNotExpanded, "~", currentu.HomeDir, -1)
	tmpRcloneConfig := fmt.Sprintf("%s/temp_rclone.config", tmpDir)

	updateConfig(getRcloneSetting(s3auth), tmpRcloneConfig, carefullUpdate)
	remoteName := getPrivateRcloneRemoteName(s3auth.projectId)
	info, err := ValidateRemote(tmpRcloneConfig, remoteName, "rclone", ValidateRcloneRemote, printTempConfigInfo, skipValidation)
	if err != nil {
		return info, err
	}
	inf, err := commitTempConfigFile(tmpRcloneConfig, rcloneConfigPath)

	if err != nil {

		return fmt.Sprintf("While updating configuration, %s", inf), err
	}

	fmt.Printf("Updated rclone config %s\n\n", rcloneConfigPath)
	fmt.Printf(passedRcloneRemoteValdidationMessage, remoteName, s3auth.projectId, remoteName, s3auth.projectId, s3auth.projectId)
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

	toolList := get_tools([]string{"rclone", "s3cmd"})
	syscall.Umask(0)

	err := parseCommandlineArguments(&programArgs)
	if err != nil {
		PrintErr(err, "Invalid input for some commandline arguments")
		os.Exit(1)
	}
	if programArgs.skipValidation {
		fmt.Printf("\n%s\n", skipValidationWarning)
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
	if programArgs.skipRcloneConfiguration {
		fmt.Printf("User gave --skip-rclone, will not configure for rclone")

	} else {
		fmt.Printf("\n=========== CONFIGURING RCLONE ===========\n")
		extraInfo, err = addRcloneRemotes(authInfo, programArgs.rcloneConfig, tmpDir, programArgs.skipValidation, programArgs.debug)
		if err != nil {
			if !toolList["rclone"] {
				fmt.Print("WARNING: rclone command is missing (if rclone is a shell alias this script will not find it)\n")
			}
			PrintErr(err, extraInfo)
		}
	}
	if programArgs.skipS3cmdConfiguration {
		fmt.Printf("User gave --skip-s3cmd, will not configure for s3cmd")
	} else {
		fmt.Printf("\n=========== CONFIGURING S3cmd ===========\n")
		extraInfo, err = adds3cmdRemote(authInfo, programArgs.s3cmdConfig, tmpDir, programArgs.skipValidation, programArgs.debug, programArgs.keepDefaultS3cmdConfig)
		if err != nil {
			if !toolList["s3cmd"] {
				fmt.Print("WARNING: s3cmd command is missing (if s3cmd is a shell alias this script will not find it)\n")
			}
			PrintErr(err, extraInfo)
		}
	}
	if !programArgs.debug {
		os.RemoveAll(tmpDir)
	}
}
