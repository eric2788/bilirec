package file

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("service", "file")

var ErrFileNotFound = fmt.Errorf("file not found")
var ErrInvalidFilePath = fmt.Errorf("invalid file path")
var ErrAccessDenied = fmt.Errorf("access denied")
var ErrIsDirectory = fmt.Errorf("path is a directory")

type Service struct {
	cfg *config.Config
}

type Tree struct {
	Name  string
	IsDir bool
	Path  string
	Size  int64
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) ListTree(path string) ([]*Tree, error) {
	return s.ListTreeWithFilter(path, func(fs.DirEntry) bool { return true })
}

func (s *Service) ListTreeWithFilter(path string, filter func(fs.DirEntry) bool) ([]*Tree, error) {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	relativePath := strings.TrimPrefix(fullPath, s.cfg.OutputDir)
	relativePath = strings.TrimPrefix(relativePath, string(os.PathSeparator))

	var files []*Tree
	for _, entry := range entries {
		if filter(entry) {
			entryPath := filepath.Join(relativePath, entry.Name())
			files = append(files, &Tree{
				Name:  entry.Name(),
				IsDir: entry.IsDir(),
				Path:  entryPath,
				Size: func() int64 {
					if info, err := entry.Info(); err == nil {
						return info.Size()
					}
					return 0
				}(),
			})
		}
	}
	return files, nil
}

func (s *Service) GetFileStream(path string) (*os.File, error) {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, ErrIsDirectory
	}

	return os.Open(fullPath)
}

func (s *Service) validatePath(path string) (string, error) {
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
