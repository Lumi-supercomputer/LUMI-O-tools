package main

type validationFunc func(string, string) error

var tools = [3]ToolSettings{rcloneSettings, s3cmdSettings, awsSettings}

const systemDefaultRcloneConfig = "~/.config/rclone/rclone.conf"
const systemDefaultS3cmdConfig = "~/.s3cfg"
const systemDefaultAwsConfig = "~/.aws/credentials"
const systemDefaultS3Url = "https://lumidata.eu"

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

const authInstructions = `Please login to  https://auth.lumidata.eu/
In the web interface, choose first the project you wish to use.
Next generate a new key or use existing valid key
Open the Key details view and based on that give following information`

type remoteNameFunc func(int) string
type addRemote func(s3auth AuthInfo, tmpDir string, debug bool, toolsettings ToolSettings) (string, error)

type Settings struct {
	chunksize       int
	debug           bool
	projectId       int
	skipValidation  string
	rcloneConfig    string
	s3cmdConfig     string
	awsCredentials  string
	configuredTools string
	nonInteractive  bool
	deleteList      string
	keepDefault     string
	url             string
}
type AuthInfo struct {
	s3AccessKey string
	s3SecretKey string
	projectId   int
	chunksize   int
	url         string
}
type ToolSettings struct {
	configPath         string
	addRemote          addRemote
	name               string
	isEnabled          bool
	isPresent          bool
	validationDisabled bool
	noReplace          bool
	carefullUpdate     bool
	singleSection      bool
}

var rcloneSettings = ToolSettings{
	configPath:         systemDefaultRcloneConfig,
	addRemote:          addRcloneRemotes,
	name:               "rclone",
	isEnabled:          true,
	isPresent:          false,
	validationDisabled: false,
	noReplace:          false,
	carefullUpdate:     true,
	singleSection:      false}
var s3cmdSettings = ToolSettings{
	configPath:         systemDefaultS3cmdConfig,
	addRemote:          adds3cmdRemote,
	name:               "s3cmd",
	isEnabled:          true,
	isPresent:          false,
	validationDisabled: false,
	noReplace:          false,
	carefullUpdate:     true,
	singleSection:      true}

var awsSettings = ToolSettings{
	configPath:         systemDefaultAwsConfig,
	addRemote:          addAwsEndPoint,
	name:               "aws",
	isEnabled:          false,
	isPresent:          false,
	validationDisabled: false,
	noReplace:          false,
	carefullUpdate:     true,
	singleSection:      false,
}
