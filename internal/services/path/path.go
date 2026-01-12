package path

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("service", "path")

var ErrFileNotFound = fmt.Errorf("file not found")
var ErrInvalidFilePath = fmt.Errorf("invalid file path")
var ErrAccessDenied = fmt.Errorf("access denied")

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		cfg: cfg,
	}
}

func (s *Service) ValidatePath(path string) (string, error) {
	baseAbs, err := filepath.Abs(s.cfg.OutputDir)
	if err != nil {
		logger.Errorf("invalid base path for %s: %v", s.cfg.OutputDir, err)
		return "", ErrInvalidFilePath
	}

	fullPath := filepath.Join(baseAbs, path)
	fullPath = filepath.Clean(fullPath)

	fullPathAbs, err := filepath.Abs(fullPath)
	if err != nil {
		logger.Errorf("invalid path for %s: %v", fullPath, err)
		return "", ErrInvalidFilePath
	}

	if !strings.HasPrefix(fullPathAbs, baseAbs+string(os.PathSeparator)) &&
		fullPathAbs != baseAbs {
		logger.Errorf("path traversal detected: %s", fullPath)
		return "", ErrAccessDenied
	}

	return fullPathAbs, nil
}

func (s *Service) GetRelativePath(fullPath string) (string, error) {
	baseAbs, err := filepath.Abs(s.cfg.OutputDir)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(baseAbs, fullPath)
	if err != nil {
		return "", err
	}
	if rel == "." {
		rel = ""
	}
	return rel, nil
}
