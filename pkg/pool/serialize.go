package pool

import (
	"encoding/gob"
)

type Serializer struct {
	encPool *BufferPool
	decPool *BufferPool
}

func NewSerializer() *Serializer {
	return &Serializer{
		encPool: NewBufferPool(1024, 64*1024),
		decPool: NewBufferPool(1024, 64*1024),
	}
}

func (s *Serializer) Serialize(v interface{}) ([]byte, error) {
	buf := s.encPool.Get()
	buf.Reset()
	enc := gob.NewEncoder(buf)
	defer s.encPool.Put(buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := append([]byte(nil), buf.Bytes()...) // copy out
	return out, nil
}

func (s *Serializer) Deserialize(data []byte, v any) error {
	buf := s.decPool.Get()
	buf.Reset()
	defer s.decPool.Put(buf)
	if _, err := buf.Write(data); err != nil {
		return err
	}
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}
