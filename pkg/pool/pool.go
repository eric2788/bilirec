package pool

import (
	"sync"
)

var DefaultBufferPool = NewBufferPool(102400)

type BufferPool sync.Pool

func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		New: func() any {
			buf := make([]byte, bufferSize)
			return &buf
		},
	}
}

func (p *BufferPool) GetBuffer() []byte {
	return *p.GetBufferPtr()
}

func (p *BufferPool) PutBuffer(buf []byte) {
	p.PutBufferPtr(&buf)
}

func (p *BufferPool) GetBufferPtr() *[]byte {
	return (*sync.Pool)(p).Get().(*[]byte)
}

func (p *BufferPool) PutBufferPtr(buf *[]byte) {
	*buf = (*buf)[:cap(*buf)]
	(*sync.Pool)(p).Put(buf)
}
