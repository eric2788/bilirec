package config

import (
	"net/url"
	"os"

	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
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

	FrontendURL  *url.URL
	Username     string
	PasswordHash string
	JwtSecret    string

	Debug          bool
	ProductionMode bool
}

func provider() (*Config, error) {

	// parse username and password

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

	// parse frontend url

	url, err := url.Parse(utils.EmptyOrElse(os.Getenv("FRONTEND_URL"), "http://localhost:8080"))

	if err != nil {
		return nil, err
	}

	// parse debug

	debug := os.Getenv("DEBUG") == "true"

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
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
		FrontendURL:             url,
		Username:                username,
		PasswordHash:            string(passwordHash),
		JwtSecret:               utils.EmptyOrElse(os.Getenv("JWT_SECRET"), "bilirec_secret"),
		Debug:                   debug,
		ProductionMode:          os.Getenv("PRODUCTION_MODE") == "true",
	}, nil
}

var Module = fx.Module("config", fx.Provide(provider))
