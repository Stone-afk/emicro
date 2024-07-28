package gzip

import (
	"bytes"
	"compress/gzip"
	"emicro/v5/rpc/compress"
	"io"
	"io/ioutil"
)

var _ compress.Compressor = Compressor{}

// Compressor implements the Compressor interface
type Compressor struct{}

func (_ Compressor) Compress(data []byte) ([]byte, error) {
	// res := &bytes.Buffer{}
	res := bytes.NewBuffer(nil)
	w := gzip.NewWriter(res)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Flush()
	if err != nil {
		return nil, err
	}
	// Defer cannot be used here. You must call Close manually
	// Otherwise, some data has not been refreshed to res,
	// execute Uncompress return []byte{}
	// This is a very error prone place
	if err = w.Close(); err != nil {
		return nil, err
	}
	return res.Bytes(), nil
}

func (_ Compressor) UnCompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.Close()
	}()
	res, err := ioutil.ReadAll(r)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return res, nil
}

func (_ Compressor) Code() byte {
	return 1
}
