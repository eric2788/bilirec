package pool

import (
	"bytes"
	"encoding/gob"
)

type Serializer struct {
	encBuf  *bytes.Buffer
	decBuf  *bytes.Buffer
	encoder *gob.Encoder
	decoder *gob.Decoder
}

func NewSerializer() *Serializer {
	encBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	decBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	return &Serializer{
		encBuf:  encBuf,
		decBuf:  decBuf,
		encoder: gob.NewEncoder(encBuf),
		decoder: gob.NewDecoder(decBuf),
	}
}

func (s *Serializer) Serialize(v interface{}) ([]byte, error) {
	s.encBuf.Reset()
	if err := s.encoder.Encode(v); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.encBuf.Bytes()...), nil
}

func (s *Serializer) Deserialize(data []byte, v any) error {
	s.decBuf.Reset()
	s.decBuf.Write(data)
	return s.decoder.Decode(v)
}
