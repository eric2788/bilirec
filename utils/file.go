package utils

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eric2788/bilirec/pkg/pool"
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

// StreamToFile streams data from rc to outPath using the provided BytesPool for buffers.
// It writes to a temp file in the same directory and atomically renames on success.
// The function closes rc before returning.
func StreamToFile(ctx context.Context, rc io.ReadCloser, outPath string, bp *pool.BytesPool) error {
	defer rc.Close()

	dir := filepath.Dir(outPath)
	tmp, err := os.CreateTemp(dir, "download-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	// buffered writer to reduce syscalls
	writer := bufio.NewWriterSize(tmp, 256*1024) // 256KB write buffer

	// get pooled buffer
	buf := bp.GetBytes()
	defer bp.PutBytes(buf)

	// perform copy in goroutine so we can observe ctx cancellation
	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.CopyBuffer(writer, rc, buf)
		if err == nil {
			// flush and sync once
			if err = writer.Flush(); err == nil {
				err = tmp.Sync()
			}
		}
		copyErrCh <- err
	}()

	select {
	case <-ctx.Done():
		// Request cancellation: closing rc usually makes io.Copy return
		_ = rc.Close()
		<-copyErrCh // wait for copy goroutine to finish / return
		cleanup()
		return ctx.Err()
	case err := <-copyErrCh:
		if err != nil {
			cleanup()
			return err
		}
	}

	// finalize
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	// remove existing target (Windows may block rename), ignore errors
	_ = os.Remove(outPath)
	if err := os.Rename(tmpName, outPath); err != nil {
		cleanup()
		return err
	}

	return nil
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
