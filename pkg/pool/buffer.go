package pool

import (
	"bytes"
	"sync"
)

type BufferPool struct {
	pool   sync.Pool
	maxCap int
}

func NewBufferPool(initialCap, maxCap int) *BufferPool {
	if initialCap < 0 {
		initialCap = 0
	}
	if maxCap < initialCap {
		maxCap = initialCap
	}
	return &BufferPool{
		pool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, initialCap))
			},
		},
		maxCap: maxCap,
	}
}

func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.pool.Get().(*bytes.Buffer)
}

func (bp *BufferPool) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	// Only pool buffers that haven't grown too large
	if buf.Cap() <= bp.maxCap {
		buf.Reset()
		bp.pool.Put(buf)
	}
	// Otherwise, let it be garbage collected
}
