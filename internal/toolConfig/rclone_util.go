package toolConfig

import (
	"fmt"
	"lumioconf/internal/util"
	"os"
	"os/user"
	"strings"
)

const passedRcloneRemoteValdidationMessage = `rclone remote %s: now provides an S3 based connection to Lumi-O storage area of project_%d

rclone remote %s: now provides an S3 based connection to Lumi-O storage area of project_%d
	Data pushed here is publicly available using the URL: https://%d.lumidata.eu/<bucket_name>/<object>"
`

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

func ValidateRcloneRemote(rcloneConfigFilePath string, remoteName string) error {
	os.Setenv("RCLONE_CONFIG", rcloneConfigFilePath)
	command_args := fmt.Sprintf("%s:", remoteName)
	return util.CheckCommand("rclone", "lsd",
		"--contimeout", "2s",
		"--timeout", "2s",
		"--low-level-retries", "1",
		"--retries", "1",
		command_args)
}

func addRcloneRemotes(s3auth AuthInfo, tmpDir string, rcloneSettings ToolSettings) (string, error) {
	currentu, _ := user.Current()
	rcloneConfigPath := strings.Replace(rcloneSettings.configPath, "~", currentu.HomeDir, -1)
	tmpRcloneConfig := fmt.Sprintf("%s/temp_rclone.config", tmpDir)
	util.UpdateConfig(getRcloneSetting(s3auth), rcloneConfigPath, tmpRcloneConfig, rcloneSettings.carefullUpdate, rcloneSettings.singleSection)
	remoteName := getPrivateRcloneRemoteName(s3auth.ProjectId)
	info, err := ValidateRemote(tmpRcloneConfig, remoteName, "rclone", ValidateRcloneRemote, rcloneSettings.ValidationDisabled)
	if err != nil {
		return info, err
	}
	inf, err := util.CommitTempConfigFile(tmpRcloneConfig, rcloneConfigPath)

	if err != nil {

		return fmt.Sprintf("While updating configuration, %s", inf), err
	}

	fmt.Printf("Updated rclone config %s\n\n", rcloneConfigPath)
	fmt.Printf(passedRcloneRemoteValdidationMessage, remoteName, s3auth.ProjectId, getPublicRcloneRemoteName(s3auth.ProjectId), s3auth.ProjectId, s3auth.ProjectId)
	return "", nil
}

func getRcloneSetting(a AuthInfo) map[string]map[string]string {
	rcloneSettings := make(map[string]map[string]string)
	privateRemoteName := getPrivateRcloneRemoteName(a.ProjectId)
	publicRemoteName := getPublicRcloneRemoteName(a.ProjectId)
	sharedRemoteSettings := map[string]string{
		"type":              "s3",
		"provider":          "Ceph",
		"env_auth":          "false",
		"project_id":        fmt.Sprintf("%d", a.ProjectId),
		"access_key_id":     a.s3AccessKey,
		"secret_access_key": a.s3SecretKey,
		"endpoint":          a.Url}
	rcloneSettings[privateRemoteName] = util.MergeMaps(map[string]string{"acl": "private"}, sharedRemoteSettings)
	rcloneSettings[publicRemoteName] = util.MergeMaps(map[string]string{"acl": "public"}, sharedRemoteSettings)

	return rcloneSettings
}
