package pool

import (
	"context"
	"errors"
	"io"
)

type PoolWriter struct {
	read  io.ReadCloser
	write io.WriteCloser
	pool  *BytesPool
}

func NewPoolWriter(read io.ReadCloser, write io.WriteCloser, poolSize int) *PoolWriter {
	return &PoolWriter{
		read:  read,
		write: write,
		pool:  NewBytesPool(poolSize),
	}
}

func (w *PoolWriter) ReadChunk(ctx context.Context, chunkSize int) (<-chan []byte, error) {
	if w.read == nil {
		return nil, errors.New("nil reader")
	}
	ch := make(chan []byte, 10) // small channel buffer to smooth bursts
	go func() {
		defer close(ch)
		defer w.read.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			buf := w.pool.GetBytes()
			// determine read length
			readLen := chunkSize
			if readLen <= 0 || readLen > cap(buf) {
				readLen = cap(buf)
			}

			n, err := w.read.Read(buf[:readLen])
			if n > 0 {
				// send the populated slice (len = n, cap = cap(buf))
				select {
				case ch <- buf[:n]:
					// consumer will call Flush to return buffer
				case <-ctx.Done():
					// caller cancelled while we were sending; return the buffer
					w.pool.PutBytes(buf)
					return
				}
			} else {
				// no bytes were read; return buffer
				w.pool.PutBytes(buf)
			}

			if err != nil {
				if err == io.EOF {
					return
				}
				return
			}
		}
	}()

	return ch, nil
}
