// Generic utility functions
package util

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

// Global flag for descided if we should print debug info
// Set when parsing commandline arguments
var GlobalDebugFlag = false

// These build info values are filled during compilation See the makefile
var (
	progVersion = ""
	gitHash     = ""
	buildTime   = ""
	isDirty     = ""
)

func PrintVersion() {
	var programName = filepath.Base(os.Args[0])
	time := "(No build time information)"
	hash := "(No commit info)"
	dirty := ""

	version := progVersion
	if isDirty != "" {
		dirty = fmt.Sprintf(" %s", "(dirty)")
	}
	if buildTime != "" {
		time = buildTime
	}
	if gitHash != "" {
		hash = gitHash
	}
	fmt.Printf("%s %s\n\tcommit: %s%s\n\tbuild time: %s\n", programName, version, hash, dirty, time)
}

func PrintVerb(msg string) {
	if GlobalDebugFlag {
		fmt.Print(msg)
	}
}
func RemoveStringFromSlice(o []string, r string) []string {
	s := make([]string, len(o))
	copy(s, o)
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func GetMaxOptionLength() int {
	maxL := 0
	flag.VisitAll(func(f *flag.Flag) {
		if len(f.Name) > maxL {
			maxL = len(f.Name)
		}
	})
	return maxL
}

func SetCustomHelp() {

	var usage = `Usage: %s [OPTIONS]

Command line tool to configure programs like rclone,s3cmd, aws cli and boto3 
to use the LUMI-O object storage
    
options:
   -h, --help%sShow this help message and exit`

	maxOptionLen := GetMaxOptionLength()
	flag.Usage = func() {
		fmt.Printf(usage+"\n", filepath.Base(os.Args[0]), strings.Repeat(" ", maxOptionLen-len("-h, --help")+4))
	}
	flagF :=
		func(ff *flag.Flag) {
			padding := maxOptionLen - len(ff.Name) + 2
			usage = usage + "\n   --" + ff.Name + strings.Repeat(" ", padding) + ff.Usage

		}
	flag.VisitAll(flagF)
}

func UpdateConfig(config map[string]map[string]string, oldConfigFilePath string, newConfigFilePath string, carefull bool, singleSectionOnly bool) (string, error) {
	_, err := os.Create(newConfigFilePath)
	if err != nil {
		return "Failed file creation", err
	}
	err = os.Chmod(newConfigFilePath, 0600)
	if err != nil {
		return "Failed chmod", err
	}
	info, err := CommitTempConfigFile(oldConfigFilePath, newConfigFilePath)
	if err != nil {
		return info, err
	}
	remoteConfig := ini.Empty()

	for sectionName, m := range config {
		for k, v := range m {
			_, err = remoteConfig.Section(sectionName).NewKey(k, v)
			if err != nil {
				return "Failed adding ini section", err
			}
		}
	}

	// Do not delete remote config before setting new values.
	if carefull {
		err = updateIniSections(newConfigFilePath, remoteConfig, singleSectionOnly)
	} else {
		err = setIniSections(newConfigFilePath, remoteConfig, singleSectionOnly)
	}
	if err != nil {
		return "Failed while editing ini sections", err
	}
	return "", nil
}
func RemoveWhiteSpaceAndSplit(a string) []string {
	reg, _ := regexp.Compile(`\s+`)
	return strings.Split(reg.ReplaceAllString(a, ""), ",")
}

func DeleteIniSectionsFromFile(filename string, sectionNames []string) error {
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return err
	}
	cfg := ini.Empty()
	err := cfg.Append(filename)
	if err != nil {
		return err
	}
	var original_value = ""
	if cfg.HasSection("default") {
		df, _ := cfg.GetSection("default")
		if df.HasKey("original_name") {
			original_value = df.Key("original_name").String()
		} else {
			fmt.Print("WARNING: Found default section but could not guess the related section\n")
		}
	}
	for _, name := range sectionNames {
		if cfg.HasSection(name) {
			cfg.DeleteSection(name)
			fmt.Printf("Deleted section %s in file %s\n", name, filename)
		} else {
			fmt.Printf("WARNING: While deleting section %s in file %s, no such section\n", name, filename)
		}
		if original_value == name {
			if cfg.HasSection("default") {
				cfg.DeleteSection("default")
				fmt.Print("WARNING: Also deleted default section\n")
			}
		}

	}
	err = cfg.SaveTo(filename)
	return err

}

func IsDirectory(path string) bool {
	currentu, _ := user.Current()
	fileInfo, err := os.Stat(strings.Replace(path, "~", currentu.HomeDir, 1))
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

func updateIniSections(filename string, data *ini.File, singleSection bool) error {
	return modifySections(filename, data, false, singleSection)
}
func modifySections(filename string, data *ini.File, setSection bool, oneSectionOnly bool) error {
	cfg := ini.Empty()
	err := cfg.Append(filename)
	if err != nil {
		return err
	}
	if oneSectionOnly {
		for _, sectionName := range cfg.SectionStrings() {
			if !StringInSlice(sectionName, data.SectionStrings()) {
				cfg.DeleteSection(sectionName)
			}
		}
	}
	for _, sectionName := range data.SectionStrings() {

		if cfg.HasSection(sectionName) && setSection {
			cfg.DeleteSection(sectionName)
		}
		_, err := cfg.NewSection(sectionName)
		if err != nil {
			return err
		}
		section, _ := cfg.GetSection(sectionName)
		section2, _ := data.GetSection(sectionName)
		for _, key := range section2.KeyStrings() {
			_, err = section.NewKey(key, section2.Key(key).String())
			if err != nil {
				return err
			}
		}
	}
	err = cfg.SaveTo(filename)
	return err
}

func setIniSections(filename string, data *ini.File, singleSection bool) error {
	return modifySections(filename, data, true, singleSection)
}

func PrintErr(err error, info string) {
	var programName = filepath.Base(os.Args[0])
	message := "%s: %s\n"
	if err != nil {
		message = "%s: %s\n\t%s\n"
		fmt.Printf(message, programName, info, err.Error())
	} else {
		fmt.Printf(message, programName, info)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func CreateTmpDir(path string) (string, error) {
	usern, _ := user.Current()

	tmpdirPath := ""
	tmpVal := path

	if tmpVal == "" {
		tmpVal = os.Getenv("TMPDIR")
	}
	if tmpVal != "" {
		tmpdirPath = fmt.Sprintf("%s/%s/lumio-temp-%s", tmpVal, usern.Username, randStringRunes(10))
	} else {
		tmpdirPath = fmt.Sprintf("/tmp/%s/lumio-temp-%s", usern.Username, randStringRunes(10))
	}
	err := os.MkdirAll(tmpdirPath, 0700)
	if err != nil {
		return "", fmt.Errorf("failed creatig tmpdir at %s, error was: %s", tmpdirPath, err.Error())
	}

	return tmpdirPath, nil
}

func CheckCommand(command string, args ...string) error {
	// Captures both stderr and stdout
	ret, err := exec.Command(command, args...).CombinedOutput()
	if err != nil {
		if len(ret) == 0 {
			return err
		} else {
			return errors.New(string(ret))
		}

	} else {
		return nil
	}

}

func ReplaceInFile(path string, pattern *regexp.Regexp, replacement string) {
	read, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	newContents := pattern.ReplaceAllString(string(read), replacement)

	err = os.WriteFile(path, []byte(newContents), 0)
	if err != nil {
		panic(err)
	}
}
func CheckFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(error, os.ErrNotExist)
}

func CommitTempConfigFile(src string, dest string) (string, error) {
	_, err := os.Stat(dest)
	if err != nil {
		err = os.MkdirAll(filepath.Dir(dest), 0700)
		if err != nil {

			return fmt.Sprintf("Failed creating %s", filepath.Dir(dest)), err
		}

		f, err := os.Create(dest)
		if err != nil {
			return fmt.Sprintf("Failed creating %s", dest), err
		}
		f.Close()
		err = os.Chmod(dest, 0600)
		if err != nil {
			return fmt.Sprintf("failed chmod on %s", dest), err
		}

	}
	_, err = os.Stat(src)
	// Source does not exists -> First time user is configuring the tool
	if err != nil {
		PrintVerb("No old configuration exists, starting from a blank slate\n")			
	} else {
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Sprintf("Failed reading temporary config %s", src), err
		}
		err = os.WriteFile(dest, data, 0600)
		if err != nil {
			return fmt.Sprintf("Failed writing new config %s", dest), err
		}
	}
	return "", nil
}

func MergeMaps[K comparable, V any](m1, m2 map[K]V) map[K]V {
	merged := make(map[K]V)

	for k, v := range m1 {
		merged[k] = v
	}

	for k, v := range m2 {
		merged[k] = v
	}

	return merged
}
