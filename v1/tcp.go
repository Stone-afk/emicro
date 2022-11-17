//go:build v1

package emicro

import (
	"emicro/v1/internal/errs"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

// TODO(Protocol, Header)
const lenBytes = 8

func ReadMsg(conn net.Conn) (bs []byte, err error) {
	msgLenBytes := make([]byte, lenBytes)
	length, err := conn.Read(msgLenBytes)
	// 捕获 panic
	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New(fmt.Sprintf("%v", msg))
		}
	}()
	if err != nil {
		return nil, err
	}
	if length != lenBytes {
		return nil, errs.ReadLenDataError
	}
	dataLen := binary.BigEndian.Uint64(msgLenBytes)
	bs = make([]byte, dataLen)
	_, err = io.ReadFull(conn, bs)
	return bs, err
}

func EncodeMsg(msg []byte) []byte {
	encode := make([]byte, lenBytes+len(msg))
	binary.BigEndian.PutUint64(encode[:lenBytes], uint64(len(msg)))
	copy(encode[lenBytes:], msg)
	return encode
}
