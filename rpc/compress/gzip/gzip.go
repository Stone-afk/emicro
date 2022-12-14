package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
)

// GzipCompressor implements the Compressor interface
type GzipCompressor struct {
}

func (_ GzipCompressor) Code() byte {
	return 1
}

// Compress data
func (_ GzipCompressor) Compress(data []byte) ([]byte, error) {
	// res := &bytes.Buffer{}
	res := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(res)
	_, err := gw.Write(data)
	if err != nil {
		return nil, err
	}
	err = gw.Flush()
	if err != nil {
		return nil, err
	}
	// Defer cannot be used here. You must call Close manually
	// Otherwise, some data has not been refreshed to res,
	// execute Uncompress return []byte{}
	// This is a very error prone place
	if err = gw.Close(); err != nil {
		return nil, err
	}

	return res.Bytes(), nil
}

// Uncompress data
func (_ GzipCompressor) Uncompress(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = gr.Close()
	}()
	return io.ReadAll(gr)
}
