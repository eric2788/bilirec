package stream

import (
	"context"
	"io"

	"github.com/eric2788/bilirec/pkg/pool"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("service", "stream")

type Service struct {
	pool *pool.BufferPool
}

func NewService() *Service {
	return &Service{pool: pool.DefaultBufferPool}
}

func (r *Service) ReadStream(resp *resty.Response, ctx context.Context) (<-chan []byte, error) {
	ch := make(chan []byte, 50) // 50 MB buffer
	go r.read(ch, resp.RawBody(), ctx)
	return ch, nil
}

func (r *Service) Flush(buf []byte) {
	r.pool.PutBuffer(buf)
}

func (r *Service) read(ch chan<- []byte, stream io.ReadCloser, ctx context.Context) {
	defer stream.Close()
	defer close(ch)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf := r.pool.GetBuffer()
			n, err := stream.Read(buf)
			if err == io.EOF {
				logger.Info("stream ended")
				r.Flush(buf)
				return
			} else if err != nil {
				logger.Errorf("error reading stream: %v", err)
				r.Flush(buf)
				return
			}
			if n > 0 {
				ch <- buf[:n]
			} else {
				r.Flush(buf)
			}
		}
	}
}
