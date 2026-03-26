package main

import (
	_ "embed"
	"os"

	"github.com/joho/godotenv"
)

//go:embed .env
var dotEnvFile string

func init() {

	if _, err := os.Stat("/.dockerenv"); err == nil {
		logger.Debug("running in Docker, skipping .env generation")
		return
	}

	// 生成的 .env 文件会放在可执行文件同目录下，方便用户修改
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		if err := os.WriteFile(".env", []byte(dotEnvFile), 0644); err != nil {
			logger.Warnf("failed to write .env file: %v, please create .env file manually if you are using binary", err)
		} else {
			logger.Info("generated .env file with default values, please restart each time you configure the .env file if you are using binary")
		}
	}

	if err := godotenv.Load(); err != nil {
		logger.Warnf("failed to load .env file: %v, please restart if you are using binary", err)
	} else {
		logger.Info("loaded environment variables from .env file")
	}

}
