package file

import (
	"path/filepath"

	"github.com/shirou/gopsutil/v4/disk"
)

type DiskSpace struct {
	Used  uint64 `json:"used"`
	Total uint64 `json:"total"`
	Free  uint64 `json:"free"`
}

// GetDiskSpace returns disk usage information for the output directory
func (s *Service) GetDiskSpace() (*DiskSpace, error) {
	// Get the absolute path of the output directory
	fullPath, err := filepath.Abs(s.cfg.OutputDir)
	if err != nil {
		return nil, err
	}

	// Get disk usage statistics for the path
	usage, err := disk.Usage(fullPath)
	if err != nil {
		return nil, err
	}

	return &DiskSpace{
		Used:  usage.Used,
		Total: usage.Total,
		Free:  usage.Free,
	}, nil
}
