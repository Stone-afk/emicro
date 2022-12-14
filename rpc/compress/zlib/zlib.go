package zlib

import (
	"bytes"
	"compress/zlib"
	"io"
	"io/ioutil"
)

// ZlibCompressor implements the Compressor interface
type ZlibCompressor struct {
}

func (_ ZlibCompressor) Code() byte {
	return 4
}

// Compress data
func (_ ZlibCompressor) Compress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	w := zlib.NewWriter(buf)
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
	return buf.Bytes(), err
}

// Uncompress data
func (_ ZlibCompressor) Uncompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewBuffer(data))
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
