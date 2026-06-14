package config

import (
	"os"
	"path/filepath"
	"strings"
)

var Debug = false

func ConfigExists() bool {
	path := filepath.Join(Dir(), "config.json")
	_, err := os.Stat(path)
	return err == nil
}

func Dir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".lightcode")
	os.MkdirAll(dir, 0755)
	return dir
}

func SkillsPath() string {
	if skillPath := strings.TrimSpace(os.Getenv("SKILL_PATH")); skillPath != "" {
		return skillPath
	}
	return GetCustomization().SkillsPath
}

func GetAuthPath() (string, error) {
	path := filepath.Join(Dir(), "auth.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", err
	}
	return path, nil

}

func DBPath() string {
	return filepath.Join(Dir(), "lightcode.db")
}
