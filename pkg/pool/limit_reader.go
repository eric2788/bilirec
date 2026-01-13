package pool

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

type LimitReader struct {
	r io.Reader
	l *rate.Limiter
	c context.Context
}

func NewLimitReader(ctx context.Context, r io.Reader, limit, burst int) *LimitReader {
	return &LimitReader{
		r: r,
		l: rate.NewLimiter(rate.Limit(limit), burst),
		c: ctx,
	}
}

func (lr *LimitReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	if n > 0 {
		lr.l.WaitN(lr.c, n)
	}
	return n, err
}
