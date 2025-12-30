package config

import (
	"os"

	"github.com/eric2788/bilirec/utils"
	"go.uber.org/fx"
)

// all config will be loaded from environment variables
type Config struct {
	AnonymousLogin bool
	Port           string

	MaxConcurrentRecordings int
	MaxRecordingHours       int
	MaxRecoveryAttempts     int

	OutputDir string
	SecretDir string
}

func provider() *Config {
	return &Config{
		AnonymousLogin:          os.Getenv("ANONYMOUS_LOGIN") == "true",
		Port:                    utils.EmptyOrElse(os.Getenv("PORT"), "8080"),
		MaxConcurrentRecordings: utils.MustAtoi(utils.EmptyOrElse(os.Getenv("MAX_CONCURRENT_RECORDINGS"), "3")),
		MaxRecordingHours:       utils.MustAtoi(utils.EmptyOrElse(os.Getenv("MAX_RECORDING_HOURS"), "5")),
		MaxRecoveryAttempts:     utils.MustAtoi(utils.EmptyOrElse(os.Getenv("MAX_RECOVERY_ATTEMPTS"), "5")),
		OutputDir:               utils.EmptyOrElse(os.Getenv("OUTPUT_DIR"), "records"),
		SecretDir:               utils.EmptyOrElse(os.Getenv("SECRET_DIR"), "secrets"),
	}
}

var Module = fx.Module("config", fx.Provide(provider))
