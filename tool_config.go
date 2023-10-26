package main

type validationFunc func(string, string) error

var tools = [2]ToolSettings{rcloneSettings, s3cmdSettings}

const systemDefaultRcloneConfig = "~/.config/rclone/rclone.conf"
const systemDefaultS3cmdConfig = "~/.s3cfg"
const systemDefaultAwsConfig = "~/.aws/credentials"
const carefullUpdate = false

type remoteNameFunc func(int) string

type ToolSettings struct {
	defaultConfigPath    string
	getPrivateRemoteName remoteNameFunc
	getPublicRemoteName  remoteNameFunc
	name                 string
	validate             validationFunc
	isEnabled            bool
	isPresent            bool
	validationDisabled   bool
	noReplace            bool
}

var rcloneSettings = ToolSettings{
	defaultConfigPath:    systemDefaultRcloneConfig,
	getPrivateRemoteName: getPrivateRcloneRemoteName,
	getPublicRemoteName:  getPublicRcloneRemoteName,
	validate:             ValidateRcloneRemote,
	name:                 "rclone",
	isEnabled:            true,
	isPresent:            false,
	validationDisabled:   false,
	noReplace:            false}
var s3cmdSettings = ToolSettings{
	defaultConfigPath:    systemDefaultS3cmdConfig,
	getPrivateRemoteName: getGenericRemoteName,
	getPublicRemoteName:  getGenericRemoteName,
	validate:             ValidateS3cmdRemote,
	name:                 "s3cmd",
	isEnabled:            true,
	isPresent:            false,
	validationDisabled:   false,
	noReplace:            false}
