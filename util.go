package main

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

func stringInSlice(a string, list []string) bool {
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

func get_tools(tools []string) map[string]bool {
	toolRes := make(map[string]bool)
	for _, tool := range tools {
		_, err := exec.LookPath(tool)
		if err != nil {
			toolRes[tool] = false
		} else {
			toolRes[tool] = true
		}
	}
	return toolRes
}

func SetCustomHelp() {

	var usage = `usage: %s [OPTIONS]
    
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

func updateConfig(config map[string]map[string]string, oldConfigFilePath string, newConfigFilePath string, carefull bool, singleSectionOnly bool) {
	os.Create(newConfigFilePath)
	os.Chmod(newConfigFilePath, 0600)
	commitTempConfigFile(oldConfigFilePath, newConfigFilePath)
	remoteConfig := ini.Empty()

	for sectionName, m := range config {
		for k, v := range m {
			remoteConfig.Section(sectionName).NewKey(k, v)
		}
	}

	// Do not delete remote config before setting new values.
	if carefull {
		updateIniSections(newConfigFilePath, remoteConfig, singleSectionOnly)
	} else {
		setIniSections(newConfigFilePath, remoteConfig, singleSectionOnly)
	}
}

func deleteIniSectionsFromFile(filename string, sectionNames []string) error {
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return err
	}
	cfg := ini.Empty()
	cfg.Append(filename)
	var original_value = ""
	if cfg.HasSection("default") {
		df, _ := cfg.GetSection("default")
		if df.HasKey("original_name") {
			original_value = df.Key("original_name").String()
		} else {
			fmt.Print("WARNING: Found default section but could not guess the related section")
		}
	}
	for _, name := range sectionNames {
		if cfg.HasSection(name) {
			cfg.DeleteSection(name)
			fmt.Printf("Deleted section %s in file %s", name, filename)
		} else {
			fmt.Printf("WARNING: While deleting section %s in file %s, no such section\n", name, filename)
		}
		if original_value == name {
			if cfg.HasSection("default") {
				cfg.DeleteSection("default")
			}
		}

	}
	err := cfg.SaveTo(filename)
	return err

}

func updateIniSections(filename string, data *ini.File, singleSection bool) error {
	return modifySections(filename, data, false, singleSection)
}
func modifySections(filename string, data *ini.File, setSection bool, oneSectionOnly bool) error {
	cfg := ini.Empty()
	cfg.Append(filename)
	if oneSectionOnly {
		for _, sectionName := range cfg.SectionStrings() {
			if !stringInSlice(sectionName, data.SectionStrings()) {
				cfg.DeleteSection(sectionName)
			}
		}
	}
	for _, sectionName := range data.SectionStrings() {

		if cfg.HasSection(sectionName) && setSection {
			cfg.DeleteSection(sectionName)
		}
		cfg.NewSection(sectionName)
		section, _ := cfg.GetSection(sectionName)
		section2, _ := data.GetSection(sectionName)
		for _, key := range section2.KeyStrings() {
			section.NewKey(key, section2.Key(key).String())
		}
	}
	err := cfg.SaveTo(filename)
	return err
}

func setIniSections(filename string, data *ini.File, singleSection bool) error {
	return modifySections(filename, data, true, singleSection)
}

func PrintErr(err error, info string) {
	message := "%s: %s\n"
	if err != nil {
		message = "%s: %s\n\t%s\n"
		fmt.Printf(message, programName, info, err.Error())
	} else {
		fmt.Printf(message, programName, info)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func createTmpDir(path string) string {
	usern, _ := user.Current()

	tmpdirPath := ""
	tmpVal := path

	if tmpVal == "" {
		tmpVal = os.Getenv("TMPDIR")
	}
	if tmpVal != "" {
		tmpdirPath = fmt.Sprintf("%s/%s/lumio-temp-%s", tmpVal, usern.Username, RandStringRunes(10))
	} else {
		tmpdirPath = fmt.Sprintf("/tmp/%s/lumio-temp-%s", usern.Username, RandStringRunes(10))
	}
	os.MkdirAll(tmpdirPath, 0700)
	return tmpdirPath
}

func checkCommand(command string, args ...string) error {
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

func replaceInFile(path string, pattern *regexp.Regexp, replacement string) {
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

func commitTempConfigFile(src string, dest string) (string, error) {
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
		os.Chmod(dest, 0600)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Sprintf("Failed reading temporary config %s", src), err
	}
	os.WriteFile(dest, data, 0600)
	if err != nil {
		return fmt.Sprintf("Failed writing new config %s", dest), err
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
