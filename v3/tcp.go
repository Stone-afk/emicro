//go:build v3

package emicro

import (
	"emicro/v3/internal/errs"
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
	//  bytes -> unint32
	//  get head length
	headLength := binary.BigEndian.Uint32(lenBs[:4])
	//  get body length
	bodyLength := binary.BigEndian.Uint32(lenBs[4:8])
	// read all data
	bs = make([]byte, headLength+bodyLength)
	_, err = conn.Read(bs[lenBytes:])
	// _, err = io.ReadFull(conn, bs)
	copy(bs[:lenBytes], lenBs)
	return bs, err
}
