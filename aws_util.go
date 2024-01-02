package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

const passedAwsRemoteValdidationMessage = `Created aws credentials config profile %s for project_%d
	use the specific project with the --profile flag
`
const lumioS3serviceConfig = `[services %s]
s3           = 
  endpoint_url = %s
  multipart_chunksize = %d
`

func deleteAwsEntry(path string, sectionNames []string) {

	replaceInFile(path, regexp.MustCompile(`(?m)^\s+`), "@")
	cfg, err := ini.Load(path)
	if err == nil {
		for _, sectionName := range sectionNames {
			if cfg.HasSection(sectionName) {
				cfg.DeleteSection(sectionName)
				cfg.SaveTo(path)
			}
		}
	}
	replaceInFile(path, regexp.MustCompile(`(?m)^@`), "  ")
}
func ValidateAwsRemote(awsCredentialFilepath string, remoteName string) error {
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", awsCredentialFilepath)
	os.Setenv("AWS_CONFIG_FILE", getAwsConfigFilePath(awsCredentialFilepath))
	return checkCommand("aws", "s3", "ls", "--profile", remoteName, "--cli-read-timeout", "2", "--cli-connect-timeout", "2")
}

func getAwsConfigFilePath(pathToCredFile string) string {
	return filepath.Join(filepath.Dir(pathToCredFile), "config")
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
	if _, err := f.WriteString(fmt.Sprintf(lumioS3serviceConfig, remoteName, info.url, info.chunksize)); err != nil {
		return err
	}
	return nil
}

func getAwsSetting(a AuthInfo) map[string]map[string]string {
	awsSettings := make(map[string]map[string]string)
	// getGenericRemoteName(a.projectId)
	awsSettings[getGenericRemoteName(a.projectId)] = map[string]string{
		"aws_access_key_id":     a.s3AccessKey,
		"aws_secret_access_key": a.s3SecretKey,
		"services":              getGenericRemoteName(a.projectId),
		"project_id":            fmt.Sprintf("%d", a.projectId)}
	return awsSettings
}

func addAwsEndPoint(s3auth AuthInfo, tmpDir string, printTempConfigInfo bool, awsSettings ToolSettings) (string, error) {
	currentu, _ := user.Current()
	awsConfigPath := strings.Replace(awsSettings.configPath, "~", currentu.HomeDir, -1)
	tmpAwsConfig := fmt.Sprintf("%s/temp_aws.config", tmpDir)
	newConfig := getAwsSetting(s3auth)
	if !awsSettings.noReplace {
		newConfig["default"] = newConfig[getGenericRemoteName(s3auth.projectId)]
		newConfig["default"]["original_name"] = getGenericRemoteName(s3auth.projectId)
	}
	updateConfig(newConfig, awsConfigPath, tmpAwsConfig, awsSettings.carefullUpdate, awsSettings.singleSection)
	remoteName := getGenericRemoteName(s3auth.projectId)
	commitTempConfigFile(getAwsConfigFilePath(awsConfigPath), getAwsConfigFilePath(tmpAwsConfig))
	appendDefaultAwsEndPoint(tmpAwsConfig, s3auth, remoteName)
	info, err := ValidateRemote(tmpAwsConfig, remoteName, "aws", ValidateAwsRemote, printTempConfigInfo, awsSettings.validationDisabled)
	if err != nil {
		return info, err
	}
	inf, err := commitTempConfigFile(tmpAwsConfig, awsConfigPath)

	if err != nil {

		return fmt.Sprintf("While updating configuration, %s", inf), err
	}
	inf, err = commitTempConfigFile(getAwsConfigFilePath(tmpAwsConfig), getAwsConfigFilePath(awsConfigPath))
	if err != nil {
		return fmt.Sprintf("While setting default aws endpoint, %s", inf), err
	}

	fmt.Printf("Updated aws config %s\n\n", awsConfigPath)
	if awsSettings.noReplace {
		fmt.Printf("New config not set as default, use the --profile flag to select the generated config\n")
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
			fmt.Printf("\tNo default config set")
		}

	}
	fmt.Printf(passedAwsRemoteValdidationMessage, remoteName, s3auth.projectId)
	return "", nil
}
