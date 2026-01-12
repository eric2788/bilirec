package pool

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

type LimitReader struct {
	r io.Reader
	l *rate.Limiter
}

func NewLimitReader(r io.Reader, limit, burst int) *LimitReader {
	return &LimitReader{
		r: r,
		l: rate.NewLimiter(rate.Limit(limit), burst),
	}
}

func (lr *LimitReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	if n > 0 {
		lr.l.WaitN(context.Background(), n)
	}
	return n, err
}
