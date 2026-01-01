package config

import (
	"os"

	"github.com/eric2788/bilirec/utils"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"
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

	ConvertFLVToMp4       bool
	DeleteFlvAfterConvert bool

	Username     string
	PasswordHash string
	JwtSecret    string
}

func provider() (*Config, error) {

	password := os.Getenv("PASSWORD")
	username := os.Getenv("USERNAME")

	var passwordHash []byte
	var err error

	if password != "" && username != "" {
		passwordHash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	} else {
		passwordHash, err = []byte{}, nil
	}

	if err != nil {
		return nil, err
	}

	return &Config{
		AnonymousLogin:          os.Getenv("ANONYMOUS_LOGIN") == "true",
		Port:                    utils.EmptyOrElse(os.Getenv("PORT"), "8080"),
		MaxConcurrentRecordings: utils.MustAtoi(utils.EmptyOrElse(os.Getenv("MAX_CONCURRENT_RECORDINGS"), "3")),
		MaxRecordingHours:       utils.MustAtoi(utils.EmptyOrElse(os.Getenv("MAX_RECORDING_HOURS"), "5")),
		MaxRecoveryAttempts:     utils.MustAtoi(utils.EmptyOrElse(os.Getenv("MAX_RECOVERY_ATTEMPTS"), "5")),
		OutputDir:               utils.EmptyOrElse(os.Getenv("OUTPUT_DIR"), "records"),
		SecretDir:               utils.EmptyOrElse(os.Getenv("SECRET_DIR"), "secrets"),
		ConvertFLVToMp4:         os.Getenv("CONVERT_FLV_TO_MP4") == "true",
		DeleteFlvAfterConvert:   os.Getenv("DELETE_FLV_AFTER_CONVERT") == "true",
		Username:                username,
		PasswordHash:            string(passwordHash),
		JwtSecret:               utils.EmptyOrElse(os.Getenv("JWT_SECRET"), "bilirec_secret"),
	}, nil
}

var Module = fx.Module("config", fx.Provide(provider))
