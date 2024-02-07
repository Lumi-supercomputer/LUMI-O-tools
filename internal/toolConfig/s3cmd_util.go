package toolConfig

import (
	"errors"
	"fmt"
	"lumioconf/internal/util"
	"os"
	"os/user"
	"strings"

	"gopkg.in/ini.v1"
)

const passedS3cmdRemoteValidationMessage = `Created s3cmd config %s for project_%d
	Other existing configurations can be accessed by adding the -c flag
	s3cmd -c ~/.s3cfg-<profile-name> COMMAND ARGS
`
const noUpdates3cfgMessage = `Default s3cmd config was not chaged, current default is %s in file %s
Either set S3CMD_CONFIG
Or use the -c flag on the commandline to use the generated config

`

func ValidateS3cmdRemote(s3cmdConfigFilePath string, remoteName string) error {
	return util.CheckCommand("s3cmd", "-c", s3cmdConfigFilePath, "ls", "s3:")
}

func deleteExtraS3cmdConfig(configFile string, projectNames []string) error {
	currentu, _ := user.Current()
	if configFile == systemDefaultConfigPaths["s3cmd"] {
		for _, projectName := range projectNames {
			configFullPath := strings.Replace(configFile, "~", currentu.HomeDir, 1)
			extraConfig := fmt.Sprintf("%s-%s", configFullPath, projectName)
			if util.CheckFileExists(extraConfig) {
				fmt.Printf("Removing profile %s by deleting the file %s\n", projectName, extraConfig)
				err := os.Remove(extraConfig)
				if err != nil {
					return err
				}
			}

		}
	}
	return nil

}

func getS3cmdSetting(a AuthInfo) map[string]map[string]string {
	s3cmdSettings := make(map[string]map[string]string)
	s3cmdSettings[getGenericRemoteName(a.ProjectId)] = map[string]string{"access_key": a.s3AccessKey,
		"secret_key":           a.s3SecretKey,
		"host_base":            a.Url,
		"host_bucket":          a.Url,
		"human_readable_sizes": "True",
		"project_id":           fmt.Sprintf("%d", a.ProjectId),
		"enable_multipart":     "True",
		"signature_v2":         "True",
		"use_https":            "True",
		"chunk_size":           fmt.Sprintf("%d", a.Chunksize)}
	return s3cmdSettings

}

func adds3cmdRemote(s3auth AuthInfo, tmpDir string, s3cmdSettings ToolSettings) (string, error) {

	currentu, _ := user.Current()
	s3cmdBaseConfigPath := fmt.Sprintf("%s", strings.Replace(s3cmdSettings.configPath, "~", currentu.HomeDir, 1))
	nonDefaultConfigPathSet := s3cmdSettings.configPath != systemDefaultConfigPaths["s3cmd"]
	s3cmdConfigPath := s3cmdBaseConfigPath
	tmps3cmdConfig := fmt.Sprintf("%s/temp_s3cmd.config", tmpDir)
	remoteName := getGenericRemoteName(s3auth.ProjectId)
	util.UpdateConfig(getS3cmdSetting(s3auth), s3cmdConfigPath, tmps3cmdConfig, s3cmdSettings.carefullUpdate, s3cmdSettings.singleSection)
	info, err := ValidateRemote(tmps3cmdConfig, remoteName, "s3cmd", ValidateS3cmdRemote, s3cmdSettings.ValidationDisabled)
	if err != nil {
		return info, err
	}

	if _, err := os.Stat(s3cmdBaseConfigPath); errors.Is(err, os.ErrNotExist) {
		if s3cmdSettings.NoReplace {
			fmt.Printf("WARNING: --keep-default used with s3cmd, but %s does not exists\n", s3cmdBaseConfigPath)
		}
	}

	// For custom locations it does not make sense to have pseudo defaults.
	if !nonDefaultConfigPathSet {
		s3cmdConfigPath = fmt.Sprintf("%s-%s", s3cmdBaseConfigPath, getGenericRemoteName(s3auth.ProjectId))
		inf, err := util.CommitTempConfigFile(tmps3cmdConfig, s3cmdBaseConfigPath)
		if err != nil {
			return fmt.Sprintf("While updating configuration, %s", inf), err
		}
	}

	inf, err := util.CommitTempConfigFile(tmps3cmdConfig, s3cmdConfigPath)
	if err != nil {
		return fmt.Sprintf("While updating configuration, %s", inf), err
	}
	if !s3cmdSettings.NoReplace && !nonDefaultConfigPathSet {
		fmt.Printf("Updated s3cmd config %s\n\n", s3cmdConfigPath)
	} else {
		if s3cmdSettings.NoReplace && !nonDefaultConfigPathSet {
			fmt.Printf("Saved generated config to %s\n", s3cmdConfigPath)
			cfg, err := ini.Load(s3cmdBaseConfigPath)
			if err == nil {
				fmt.Printf(noUpdates3cfgMessage, cfg.Sections()[1].Name(), s3cmdBaseConfigPath)
			} else {
				fmt.Printf("No default configuration exists, use S3CMD_CONFIG or the -c flag to use the generated config\n")
			}
		}
	}
	if !s3cmdSettings.NoReplace {
		fmt.Printf("New configuration set as default\n")
	}
	fmt.Printf(passedS3cmdRemoteValidationMessage, remoteName, s3auth.ProjectId)
	return "", nil

}
