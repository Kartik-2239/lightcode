package config

import (
	"os"
	"path/filepath"
)

func Dir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".lightcode")
	os.MkdirAll(dir, 0755)
	return dir
}

func DBPath() string {
	return filepath.Join(Dir(), "lightcode.db")
}

func EnvPath() string {
	return filepath.Join(Dir(), ".env")
}
