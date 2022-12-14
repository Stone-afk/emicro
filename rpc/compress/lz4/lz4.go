package lz4

import (
	"github.com/pierrec/lz4/v4"
)

// Compressor lz4 zip Lz4 is a compression algorithm that allows "everyone to love and flowers to bloom".
// It can be well extended on multi-core. The compression rate of lz4 is slightly inferior,
// but it has an amazing advantage in decompression speed (about 3 times that of gzip (multiple tests)).
// Because of the efficient multi-core utilization during compression and the amazing decompression,
// lz4 has been used in many important occasions!
// Lz4 is very suitable for scenarios that require frequent compression and real-time fast decompression;
// The object extracted by lz4 is a file, not a directory.

type Compressor struct {
	lz4.Compressor
}

func (c Compressor) Code() byte {
	return 2
}

// Compress data
func (c Compressor) Compress(data []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := c.CompressBlock(data, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Uncompress data
func (c Compressor) Uncompress(data []byte) ([]byte, error) {
	// Allocate a very large buffer for decompression.
	buf := make([]byte, 10*len(data))
	n, err := lz4.UncompressBlock(data, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
