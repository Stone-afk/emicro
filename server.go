package emicro

import (
	"context"
	"emicro/internal/errs"
	"emicro/message"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
)

// Server -> tcp conn Server
type Server struct {
	services map[string]*reflectionStub
}

// NewServer instance
func NewServer() *Server {
	return &Server{
		services: make(map[string]*reflectionStub, 8),
	}
}

// RegisterService -> Service stub
func (s *Server) RegisterService(service Service) {
	s.services[service.ServiceName()] = &reflectionStub{
		s:     service,
		value: reflect.ValueOf(service),
	}
}

// Start ->
func (s *Server) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			// 可以考虑打印日志
			fmt.Printf("server: accept connection got error: %v", err)
		}
		go func() {
			if er := s.handleConn(conn); er != nil {
				// 可以考虑打印日志
				_ = conn.Close()
				return
			}
		}()
	}
}

// handleConn ->
func (s *Server) handleConn(conn net.Conn) error {
	for {
		bs, err := ReadMsg(conn)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// go func() {}
		// req := &message.Request{}
		//err = json.Unmarshal(bs, req)
		//if err != nil {
		//	return fmt.Errorf("server: unable to deserialize request, %w", err)
		//}

		req := message.DecodeReq(bs)
		resp := s.Invoke(context.Background(), req)

		//respBs, err := json.Marshal(resp)
		//if err != nil {
		//	return fmt.Errorf("server: unable to serialize response, %w", err)
		//}
		//encode := EncodeMsg(respBs)
		//encode, err := s.encodeMsg(resp)
		//if err != nil {
		//	return err
		//}

		// calculate and set the response head length
		resp.SetHeadLength()
		// calculate and set the response body length
		resp.SetBodyLength()
		encode := message.EncodeResp(resp)
		_, er := conn.Write(encode)
		if er != nil {
			return fmt.Errorf("server: sending response failed: %v", er)
		}
	}
}

//func (s *Server) encodeMsg(msg any) ([]byte, error) {
//	bs, err := json.Marshal(msg)
//	if err != nil {
//		return nil, fmt.Errorf("server: unable to serialize response, %w", err)
//	}
//	return EncodeMsg(bs), nil
//}

// Invoke -> server Invoke
func (s *Server) Invoke(ctx context.Context, req *message.Request) *message.Response {
	stub, ok := s.services[req.ServiceName]
	if !ok {
		return &message.Response{Error: []byte(errs.InvalidServiceName.Error())}
	}

	data, err := stub.Invoke(ctx, req.Method, req.Data)
	if err != nil {
		return &message.Response{Error: []byte(err.Error())}
	}
	response := &message.Response{
		Version:    req.Version,
		Compresser: req.Compresser,
		Serializer: req.Serializer,
		MessageId:  req.MessageId,
		Data:       data,
	}
	return response
}

// reflectionStub -> service stub
type reflectionStub struct {
	s     Service
	value reflect.Value
}

// Invoke -> stub execute method by reflect
func (s *reflectionStub) Invoke(ctx context.Context, methodName string, reqData []byte) ([]byte, error) {
	method := s.value.MethodByName(methodName)
	in := reflect.New(method.Type().In(1).Elem())
	err := json.Unmarshal(reqData, in.Interface())
	if err != nil {
		return nil, err
	}
	res := method.Call([]reflect.Value{reflect.ValueOf(ctx), in})
	if len(res) > 1 && res[1].Interface() != nil {
		return nil, res[1].Interface().(error)
	}
	return json.Marshal(res[0].Interface())
}
