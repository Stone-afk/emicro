package emicro

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
)

type Server struct {
	services map[string]*reflectionStub
}

func (s *Server) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			// 可以考虑打印日志
			fmt.Printf("accept connection got error: %v", err)
		}
		go func() {
			if er := s.handleConnection(conn); er != nil {
				// 这里考虑输出日志
				_ = conn.Close()
				return
			}
		}()

	}
}

func (s *Server) handleConnection(conn net.Conn) error {
	for {
		bs, err := ReadMsg(conn)
		if err != nil {
			return err
		}
		// go func() {
		req := &Request{}
		err = json.Unmarshal(bs, req)
		if err != nil {
			return err
		}
		resp, err := s.Invoke(context.Background(), req)
		if err != nil {
			return err
		}
		encode, er := s.encodeMsg(resp)
		if er != nil {
			return fmt.Errorf("encode resp failed: %w", er)
		}
		_, er = conn.Write(encode)
		if er != nil {
			return fmt.Errorf("sending response failed: %v", er)
		}
	}
}

func (s *Server) Invoke(ctx context.Context, req *Request) (*Response, error) {
	resp := &Response{}
	return resp, nil

}

type reflectionStub struct {
	s Service
}

func (s *Server) encodeMsg(msg any) ([]byte, error) {
	return nil, nil
}
