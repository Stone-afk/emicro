package emicro

import (
	"emicro/internal/errs"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// TODO(Protocol, Header)
const lenBytes = 8

func ReadMsg(conn net.Conn) (bs []byte, err error) {
	lenBs := make([]byte, lenBytes)
	length, err := conn.Read(lenBs)
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
	//  bytes -> unint64
	dataLen := binary.BigEndian.Uint64(lenBs)
	bs = make([]byte, dataLen)
	_, err = conn.Read(bs)
	// _, err = io.ReadFull(conn, bs)
	return bs, err
}

func EncodeMsg(msg []byte) []byte {
	encode := make([]byte, lenBytes+len(msg))
	//  int -> unint64 -> bytes
	binary.BigEndian.PutUint64(encode[:lenBytes], uint64(len(msg)))
	copy(encode[lenBytes:], msg)
	return encode
}
