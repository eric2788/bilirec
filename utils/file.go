package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Helper to remove invalid filename characters
func SanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		".", "_",
	)
	return replacer.Replace(name)
}

func GetPathFormat(path string) string {
	return filepath.Ext(path)[1:]
}

func ChangePathFormat(path string, newFormat string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + "." + newFormat
	}
	return path[0:len(path)-len(ext)] + "." + newFormat
}

func IsFileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}

func FFmpegAvailable() bool {
	if err := exec.Command("ffmpeg", "-h").Run(); err != nil {
		return false
	}
	return true
}
