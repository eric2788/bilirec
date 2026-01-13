package pool

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
)

type FileStreamWriter struct {
	ctx context.Context
	bp  *BytesPool
}

func NewFileStreamWriter(ctx context.Context, pool *BytesPool) *FileStreamWriter {
	return &FileStreamWriter{
		ctx: ctx,
		bp:  pool,
	}
}

// WriteToFile streams data from rc to outPath using the provided BytesPool for buffers.
// It writes to a temp file in the same directory and atomically renames on success.
// The function closes rc before returning.
func (f *FileStreamWriter) WriteToFile(rc io.ReadCloser, outPath string, writerBufferSize int) error {
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
	writer := bufio.NewWriterSize(tmp, writerBufferSize)

	// get pooled buffer
	buf := f.bp.GetBytes()
	defer f.bp.PutBytes(buf)

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
	case <-f.ctx.Done():
		// Request cancellation: closing rc usually makes io.Copy return
		_ = rc.Close()
		<-copyErrCh // wait for copy goroutine to finish / return
		cleanup()
		return f.ctx.Err()
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
