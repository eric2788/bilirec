package main

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

//go:embed docs/swagger.json
var embeddedSwagger []byte

var logger = logrus.WithField("package", "main")

func init() {
	exe, err := os.Executable()
	if err != nil {
		logger.Warnf("cannot determine executable path: %v", err)
		// fallback to cwd
		exe = "."
	}
	exeDir := filepath.Dir(exe)
	swagDir := filepath.Join(exeDir, "docs")
	swagPath := filepath.Join(swagDir, "swagger.json")

	if _, err := os.Stat(swagPath); os.IsNotExist(err) {
		if err := os.MkdirAll(swagDir, 0755); err != nil {
			logger.Warnf("failed to create docs dir: %v", err)
		}
		if err := os.WriteFile(swagPath, embeddedSwagger, 0644); err != nil {
			logger.Warnf("failed to write embedded swagger: %v", err)
		}
		logger.Infof("wrote embedded swagger to %s", swagPath)
	}
}
