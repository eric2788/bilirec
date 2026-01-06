package monitor

import "io"

type ProgressReader struct {
	r        io.ReadCloser
	read     int64 // 已讀（bytes）
	callback func(read int64)
}

func NewProgressReader(r io.ReadCloser, cb func(read int64)) *ProgressReader {
	return &ProgressReader{
		r:        r,
		callback: cb,
	}
}

func (p *ProgressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		if p.callback != nil {
			p.callback(p.read)
		}
	}
	return n, err
}

func (p *ProgressReader) Close() error {
	return p.r.Close()
}
