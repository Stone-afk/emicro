package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Compressor for gzip
type Compressor struct{}

func (c Compressor) Code() byte {
	return 1
}

// Compress data
func (c Compressor) Compress(data []byte) ([]byte, error) {
	res := &bytes.Buffer{}
	gw := gzip.NewWriter(res)
	_, err := gw.Write(data)
	if err != nil {
		return nil, err
	}
	// Defer cannot be used here. You must call Close manually
	// Otherwise, some data has not been refreshed to res,
	// This is a very error prone place
	if err = gw.Close(); err != nil {
		return nil, err
	}
	return res.Bytes(), nil
}

// Uncompress data
func (c Compressor) Uncompress(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = gr.Close()
	}()
	return io.ReadAll(gr)
}
