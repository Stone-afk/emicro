//go:build v1

package emicro

import (
	"context"
	"emicro/v1/internal/errs"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
)

type Server struct {
	services map[string]*reflectionStub
}

//func (s *Server) StartV1(address string) error {
//	ln, err := net.Listen("tcp", address)
//	if err != nil {
//		return err
//	}
//	for {
//		conn, err := ln.Accept()
//		if err != nil {
//			fmt.Printf("accept connection got error: %v", err)
//		}
//		go s.handleConnectionV1(conn)
//	}
//}

//func (s *Server) handleConnectionV1(conn net.Conn) {
//	for {
//		bytes, err := ReadMsg(conn)
//		if err != nil {
//			return
//		}
//		// go func() {
//		u := &Request{}
//		err = json.Unmarshal(bytes, u)
//		resp, er := s.Invoke(context.Background(), u)
//		if resp == nil {
//			resp = &Response{}
//		}
//		if er != nil && len(resp.Error) == 0 {
//			resp.Error = er.Error()
//		}
//		encode, er := s.encodeMsg(resp)
//		if er != nil {
//			fmt.Printf("encode resp failed: %v", er)
//			return
//		}
//		_, er = conn.Write(encode)
//		if er != nil {
//			fmt.Printf("sending response failed: %v", er)
//		}
//	}
//}

func NewServer() *Server {
	res := &Server{
		services: make(map[string]*reflectionStub, 4),
	}
	return res
}

func (s *Server) RegisterService(service Service) {
	s.services[service.ServiceName()] = &reflectionStub{
		s:     service,
		value: reflect.ValueOf(service),
	}
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
		if resp == nil {
			resp = &Response{}
		}
		if err != nil && len(resp.Error) == 0 {
			resp.Error = err.Error()
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

func (s *Server) encodeMsg(msg any) ([]byte, error) {
	respData, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return EncodeMsg(respData), nil
}

func (s *Server) Invoke(ctx context.Context, req *Request) (*Response, error) {
	stub, ok := s.services[req.ServiceName]
	if !ok {
		return nil, errs.InvalidServiceName
	}
	data, err := stub.Invoke(ctx, req.Method, req.Data)
	if err != nil {
		return nil, err
	}

	return &Response{Data: data}, nil

}

type reflectionStub struct {
	s     Service
	value reflect.Value
}

func (s *reflectionStub) Invoke(ctx context.Context, methodName string, reqData []byte) ([]byte, error) {
	method := s.value.MethodByName(methodName)
	inTyp := method.Type().In(1)
	in := reflect.New(inTyp.Elem())
	err := json.Unmarshal(reqData, in.Interface())
	if err != nil {
		return nil, err
	}
	res := method.Call([]reflect.Value{reflect.ValueOf(ctx), in})
	if len(res) > 1 && !res[1].IsZero() {
		return nil, res[1].Interface().(error)
	}
	return json.Marshal(res[0].Interface())
}
