package pool

import (
	"sync"
)

var DefaultBufferPool = NewBytesPool(32768) // 32 KB

type BytesPool sync.Pool

func NewBytesPool(bufferSize int) *BytesPool {
	return &BytesPool{
		New: func() any {
			buf := make([]byte, bufferSize)
			return &buf
		},
	}
}

func (p *BytesPool) GetBytes() []byte {
	return *p.GetBytesPtr()
}

func (p *BytesPool) PutBytes(buf []byte) {
	p.PutBytesPtr(&buf)
}

func (p *BytesPool) GetBytesPtr() *[]byte {
	return (*sync.Pool)(p).Get().(*[]byte)
}

func (p *BytesPool) PutBytesPtr(buf *[]byte) {
	*buf = (*buf)[:cap(*buf)]
	(*sync.Pool)(p).Put(buf)
}
