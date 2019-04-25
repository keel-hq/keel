package kv

import (
	"bytes"
	"encoding/gob"
	"sync"
)

var bufferPool = sync.Pool{New: allocBuffer}

func allocBuffer() interface{} {
	return &bytes.Buffer{}
}

func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func releaseBuffer(v *bytes.Buffer) {
	v.Reset()
	v.Grow(0)
	bufferPool.Put(v)
}

// Serializer - generic serializer interface
type Serializer interface {
	Encode(source interface{}) ([]byte, error)
	Decode(data []byte, target interface{}) error
}

// DefaultSerializer - returns default serializer
func DefaultSerializer() Serializer {
	return &GobSerializer{}
}

// GobSerializer - gob based serializer
type GobSerializer struct{}

// Encode - encodes source into bytes using Gob encoder
func (s *GobSerializer) Encode(source interface{}) ([]byte, error) {
	buf := getBuffer()
	defer releaseBuffer(buf)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(source)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode - decodes given bytes into target struct
func (s *GobSerializer) Decode(data []byte, target interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(target)
}
