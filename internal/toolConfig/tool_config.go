package toolConfig

type validationFunc func(string, string) error

var systemDefaultConfigPaths = map[string]string{
	"rclone": "~/.config/rclone/rclone.conf",
	"s3cmd":  "~/.s3cfg",
	"aws":    "~/.aws/credentials"}

const systemDefaultS3Url = "https://lumidata.eu"

const SkipValidationWarning = `WARNING: The --skip-validation flag was used, configurations will not be validated and could potentially be saved in an invalid state if user input is incorrect`

const failedRemoteValidationMsg = `Failed to validate new %s endpoint %s
No new endpoint was added
Double check that the correct details were entered
Run with --debug to keep the generated temporary configuration
The error was:`

const configSavedmsg = `Generated %s config has been saved to %s
IMPORTANT: When troubleshooting, DO NOT share the whole file
ONLY the info related to the specific failed endpoint %s
`

const AuthInstructions = `Please login to  https://auth.lumidata.eu/
In the web interface, choose first the project you wish to use.
Next generate a new key or use existing valid key
Open the Key details view and based on that give following information`

type remoteNameFunc func(int) string
type AddRemote func(s3auth AuthInfo, tmpDir string, toolsettings ToolSettings) (string, error)

type Settings struct {
	Chunksize      int
	ProjectId      int
	NonInteractive bool
	DeleteList     string
	Url            string
	ShowVersion    bool
}
type AuthInfo struct {
	s3AccessKey string
	s3SecretKey string
	ProjectId   int
	Chunksize   int
	Url         string
}
type ToolSettings struct {
	configPath         string
	AddRemote          AddRemote
	Name               string
	IsEnabled          bool
	IsPresent          bool
	ValidationDisabled bool
	NoReplace          bool
	carefullUpdate     bool
	singleSection      bool
}

var RcloneSettings = ToolSettings{
	configPath:         systemDefaultConfigPaths["rclone"],
	AddRemote:          addRcloneRemotes,
	Name:               "rclone",
	IsEnabled:          true,
	IsPresent:          false,
	ValidationDisabled: false,
	NoReplace:          false,
	carefullUpdate:     true,
	singleSection:      false}
var S3cmdSettings = ToolSettings{
	configPath:         systemDefaultConfigPaths["s3cmd"],
	AddRemote:          adds3cmdRemote,
	Name:               "s3cmd",
	IsEnabled:          true,
	IsPresent:          false,
	ValidationDisabled: false,
	NoReplace:          false,
	carefullUpdate:     true,
	singleSection:      true}

var AwsSettings = ToolSettings{
	configPath:         systemDefaultConfigPaths["aws"],
	AddRemote:          addAwsEndPoint,
	Name:               "aws",
	IsEnabled:          false,
	IsPresent:          false,
	ValidationDisabled: false,
	NoReplace:          false,
	carefullUpdate:     true,
	singleSection:      false,
}
