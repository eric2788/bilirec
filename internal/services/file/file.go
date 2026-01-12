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
	"github.com/eric2788/bilirec/internal/services/path"
	"go.uber.org/fx"
)

// var logger = logrus.WithField("service", "file")

var ErrIsDirectory = fmt.Errorf("path is a directory")

type Service struct {
	cfg *config.Config
	ctx context.Context

	path *path.Service
}

type Tree struct {
	Name        string `json:"name"`
	IsDir       bool   `json:"is_dir"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	IsRecording bool   `json:"is_recording,omitempty"`
}

func NewService(ls fx.Lifecycle, cfg *config.Config, pathSvc *path.Service) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		cfg:  cfg,
		ctx:  ctx,
		path: pathSvc,
	}

	ls.Append(fx.StopHook(cancel))
	return s
}

func (s *Service) ListTree(path string) ([]Tree, error) {
	return s.ListTreeWithFilter(path, func(f fs.DirEntry) bool {
		return !strings.HasSuffix(f.Name(), ".tmp") // ignore .tmp files
	})
}

func (s *Service) ListTreeWithFilter(path string, filter func(fs.DirEntry) bool) ([]Tree, error) {
	fullPath, err := s.path.ValidatePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	relativePath, err := s.path.GetRelativePath(fullPath)
	if err != nil {
		return nil, err
	}

	files := make([]Tree, 0)
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

func (s *Service) GetFileStream(path string) (io.ReadCloser, os.FileInfo, error) {
	fullPath, err := s.path.ValidatePath(path)
	if err != nil {
		return nil, nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, nil, err
	}
	if info.IsDir() {
		return nil, nil, ErrIsDirectory
	}

	if f, err := os.Open(fullPath); err != nil {
		return nil, nil, err
	} else {
		return f, info, nil
	}
}

func (s *Service) DeleteDirectory(path string) error {
	fullPath, err := s.path.ValidatePath(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(fullPath)
}

func (s *Service) DeleteFiles(paths ...string) error {
	var fullPaths []string
	for _, path := range paths {
		fullPath, err := s.path.ValidatePath(path)
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
