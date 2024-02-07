package toolConfig

import (
	"fmt"
	"lumioconf/internal/util"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

const passedAwsRemoteValdidationMessage = `Created aws credentials config profile %s for project_%d
	use a specific profile with the --profile flag
`
const lumioS3serviceConfig = `[services %s]
s3           = 
  endpoint_url = %s
  multipart_chunksize = %d
`

func deleteAwsEntry(path string, sectionNames []string) error {
	util.ReplaceInFile(path, regexp.MustCompile(`(?m)^\s+`), "@")
	cfg, err := ini.Load(path)
	if err == nil {
		for _, sectionName := range sectionNames {
			if cfg.HasSection(sectionName) {
				cfg.DeleteSection(sectionName)
				cfg.SaveTo(path)
			}
		}
	} else {
		return err
	}
	util.ReplaceInFile(path, regexp.MustCompile(`(?m)^@`), "  ")
	return nil
}

func ValidateAwsRemote(awsCredentialFilepath string, remoteName string) error {
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", awsCredentialFilepath)
	os.Setenv("AWS_CONFIG_FILE", getAwsConfigFilePath(awsCredentialFilepath))
	return util.CheckCommand("aws", "s3", "ls", "--profile", remoteName, "--cli-read-timeout", "2", "--cli-connect-timeout", "2")
}

// If we are saving the aws config file in a non standard location
// Name it in a better fashion to avoid confusion
func getAwsConfigFilePath(pathToCredFile string) string {
	currentu, _ := user.Current()
	customConfigFilePath, customPathisSet := os.LookupEnv("LUMIO_AWS_CONFIG_FILE_PATH")
	if pathToCredFile == strings.Replace(systemDefaultConfigPaths["aws"], "~", currentu.HomeDir, 1) {
		return filepath.Join(filepath.Dir(pathToCredFile), "config")

	} else if customPathisSet {
		return customConfigFilePath

	} else {
		return filepath.Join(filepath.Dir(pathToCredFile), "aws-config")

	}
}

func appendDefaultAwsEndPoint(pathToCredFile string, info AuthInfo, remoteName string) error {
	configFilePath := getAwsConfigFilePath(pathToCredFile)
	sectionName := fmt.Sprintf("services %s", remoteName)
	deleteAwsEntry(configFilePath, []string{sectionName})
	f, err := os.OpenFile(configFilePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf(lumioS3serviceConfig, remoteName, info.Url, info.Chunksize)); err != nil {
		return err
	}
	return nil
}

func getAwsSetting(a AuthInfo) map[string]map[string]string {
	awsSettings := make(map[string]map[string]string)
	// getGenericRemoteName(a.projectId)
	awsSettings[getGenericRemoteName(a.ProjectId)] = map[string]string{
		"aws_access_key_id":     a.s3AccessKey,
		"aws_secret_access_key": a.s3SecretKey,
		"services":              getGenericRemoteName(a.ProjectId),
		"project_id":            fmt.Sprintf("%d", a.ProjectId)}
	return awsSettings
}

func addAwsEndPoint(s3auth AuthInfo, tmpDir string, awsSettings ToolSettings) (string, error) {
	currentu, _ := user.Current()
	awsConfigPath := strings.Replace(awsSettings.configPath, "~", currentu.HomeDir, 1)
	tmpAwsConfig := fmt.Sprintf("%s/temp_aws.config", tmpDir)
	newConfig := getAwsSetting(s3auth)
	if !awsSettings.NoReplace {
		newConfig["default"] = newConfig[getGenericRemoteName(s3auth.ProjectId)]
		newConfig["default"]["original_name"] = getGenericRemoteName(s3auth.ProjectId)
	}
	util.UpdateConfig(newConfig, awsConfigPath, tmpAwsConfig, awsSettings.carefullUpdate, awsSettings.singleSection)
	remoteName := getGenericRemoteName(s3auth.ProjectId)
	util.CommitTempConfigFile(getAwsConfigFilePath(awsConfigPath), getAwsConfigFilePath(tmpAwsConfig))
	appendDefaultAwsEndPoint(tmpAwsConfig, s3auth, remoteName)
	info, err := ValidateRemote(tmpAwsConfig, remoteName, "aws", ValidateAwsRemote, awsSettings.ValidationDisabled)
	if err != nil {
		return info, err
	}
	inf, err := util.CommitTempConfigFile(tmpAwsConfig, awsConfigPath)

	if err != nil {

		return fmt.Sprintf("While updating configuration, %s", inf), err
	}
	inf, err = util.CommitTempConfigFile(getAwsConfigFilePath(tmpAwsConfig), getAwsConfigFilePath(awsConfigPath))
	if err != nil {
		return fmt.Sprintf("While setting default aws endpoint, %s", inf), err
	}

	fmt.Printf("Updated aws config %s\n\n", awsConfigPath)
	if awsSettings.NoReplace {
		fmt.Printf("New profile not set as default, use the --profile flag to use the generated config\n")
		cfg, err := ini.Load(awsConfigPath)
		default_config, err := cfg.GetSection("default")
		if err == nil {
			default_real_name, err := default_config.GetKey("original_name")
			if err == nil {
				fmt.Printf("\tCurrent default is %s\n", default_real_name)

			} else {
				fmt.Print("\tUnable to identify current default\n")
			}
		} else {
			fmt.Printf("\tNo default config set\n")
		}

	} else {
		fmt.Printf("New profile set as default\n")
	}
	fmt.Printf(passedAwsRemoteValdidationMessage, remoteName, s3auth.ProjectId)
	return "", nil
}
