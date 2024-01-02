package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"gopkg.in/ini.v1"
)

const passedS3cmdRemoteValidationMessage = `Created s3cmd config for project_%d
	Other existing configurations can be accessed by adding the -c flag
	s3cdm -c ~/.s3cfg-lumio-<project_number> COMMAND ARGS
`
const noUpdates3cfgMessage = `Default s3cmd config was not chaged, current default is %s in file %s
Either set S3CMD_CONFIG
Or use the -c flag on the commandline to use the generated config

`

func ValidateS3cmdRemote(s3cmdConfigFilePath string, remoteName string) error {
	return checkCommand("s3cmd", "-c", s3cmdConfigFilePath, "ls", "s3:")
}

func getS3cmdSetting(a AuthInfo) map[string]map[string]string {
	s3cmdSettings := make(map[string]map[string]string)
	s3cmdSettings[getGenericRemoteName(a.projectId)] = map[string]string{"access_key": a.s3AccessKey,
		"secret_key":           a.s3SecretKey,
		"host_base":            a.url,
		"host_bucket":          a.url,
		"human_readable_sizes": "True",
		"project_id":           fmt.Sprintf("%d", a.projectId),
		"enable_multipart":     "True",
		"signature_v2":         "True",
		"use_https":            "True",
		"chunk_size":           fmt.Sprintf("%d", a.chunksize)}
	return s3cmdSettings

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
