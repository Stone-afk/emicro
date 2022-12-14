package snappy

import (
	"bytes"
	"github.com/golang/snappy"
	"io"
	"io/ioutil"
)

// SnappyCompressor implements the Compressor interface
// SNAPPY is an open-source compression algorithm implemented by Google.
// Tests show that SNAPPY has a very high performance,
// but Google has not published a related paper.
// It can be considered that SNAPPY is an industrial algorithm.
// SNAPPY borrows ideas from LZ77. The time complexity of LZ77 matching process is too high,
// and Google has made many optimizations.
type SnappyCompressor struct {
}

func (_ SnappyCompressor) Code() byte {
	return 3
}

// Compress data
func (_ SnappyCompressor) Compress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	w := snappy.NewBufferedWriter(buf)
	_, err := w.Write(data)
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
func (_ SnappyCompressor) Uncompress(data []byte) ([]byte, error) {
	r := snappy.NewReader(bytes.NewBuffer(data))
	res, err := ioutil.ReadAll(r)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return res, nil
}

