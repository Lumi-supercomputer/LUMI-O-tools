package main

type validationFunc func(string, string) error

var tools = [2]ToolSettings{rcloneSettings, s3cmdSettings}

const systemDefaultRcloneConfig = "~/.config/rclone/rclone.conf"
const systemDefaultS3cmdConfig = "~/.s3cfg"
const systemDefaultAwsConfig = "~/.aws/credentials"

type remoteNameFunc func(int) string
type addRemote func(s3auth AuthInfo, tmpDir string, debug bool, toolsettings ToolSettings) (string, error)

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
