package file

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/processors"
	"github.com/eric2788/bilirec/pkg/pipeline"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

var logger = logrus.WithField("service", "file")

var ErrFileNotFound = fmt.Errorf("file not found")
var ErrInvalidFilePath = fmt.Errorf("invalid file path")
var ErrAccessDenied = fmt.Errorf("access denied")
var ErrIsDirectory = fmt.Errorf("path is a directory")

type Service struct {
	cfg *config.Config
	ctx context.Context
}

type Tree struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Path  string `json:"path"`
	Size  int64  `json:"size"`
}

func NewService(ls fx.Lifecycle, cfg *config.Config) *Service {
	s := &Service{cfg: cfg}
	ls.Append(fx.StartHook(func(ctx context.Context) error {
		s.ctx = ctx
		return nil
	}))
	return s
}

func (s *Service) ListTree(path string) ([]Tree, error) {
	return s.ListTreeWithFilter(path, func(fs.DirEntry) bool { return true })
}

func (s *Service) ListTreeWithFilter(path string, filter func(fs.DirEntry) bool) ([]Tree, error) {
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

	var files []Tree
	for _, entry := range entries {
		if filter(entry) {
			entryPath := filepath.Join(relativePath, entry.Name())
			files = append(files, Tree{
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

func (s *Service) GetFileStream(path, format string) (io.ReadCloser, error) {
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

	if format == "" || strings.HasSuffix(fullPath, "."+format) {
		return os.Open(fullPath)
	}

	processor := pipeline.New(processors.NewFileConverter(format))

	if err := processor.Open(s.ctx); err != nil {
		return nil, fmt.Errorf("failed to open file converter: %v", err)
	}
	defer processor.Close()

	dest, err := processor.Process(s.ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert file: %v", err)
	}

	destFile, err := os.Open(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to open converted file: %v", err)
	}

	return NewTempReader(destFile), nil
}

func (s *Service) DeleteDirectory(path string) error {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(fullPath)
}

func (s *Service) DeleteFiles(paths ...string) error {
	var fullPaths []string
	for _, path := range paths {
		fullPath, err := s.validatePath(path)
		if err != nil {
			return err
		}
		fullPaths = append(fullPaths, fullPath)
	}
	for _, fullPath := range fullPaths {
		if err := os.Remove(fullPath); err != nil {
			return err
		}
	}
	return nil
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
