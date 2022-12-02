package emicro

import (
	"context"
	"emicro/internal/errs"
	"emicro/message"
	"emicro/serialize"
	"emicro/serialize/json"
	"fmt"
	"io"
	"net"
	"reflect"
)

// Server -> tcp conn Server
type Server struct {
	services    map[string]*reflectionStub
	serializers []serialize.Serializer
}

// MustRegister -> panic error
func (s *Server) MustRegister(service Service) {
	err := s.RegisterService(service)
	if err != nil {
		panic(err)
	}
}

// NewServer instance
func NewServer() *Server {
	res := &Server{
		services: make(map[string]*reflectionStub, 8),
		// 一个字节，最多有 256 个实现，直接做成一个简单的 bit array 的东西
		serializers: make([]serialize.Serializer, 256),
	}
	// 注册最基本的序列化协议
	res.RegisterSerializer(json.Serializer{})
	return res
}

// RegisterService -> Service stub
func (s *Server) RegisterService(service Service) error {
	s.services[service.ServiceName()] = &reflectionStub{
		s:           service,
		serializers: s.serializers,
		value:       reflect.ValueOf(service),
	}
	return nil
}

// RegisterSerializer -> register serializer
func (s *Server) RegisterSerializer(serializer serialize.Serializer) {
	s.serializers[serializer.Code()] = serializer
}

// Start -> run server
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

// handleConn -> handle tcp connection
func (s *Server) handleConn(conn net.Conn) error {
	for {
		bs, err := ReadMsg(conn)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		req := message.DecodeReq(bs)
		resp := s.Invoke(context.Background(), req)

		// calculate and set the response head length
		resp.SetHeadLength()
		// calculate and set the response body length
		resp.SetBodyLength()
		encode := message.EncodeResp(resp)
		_, er := conn.Write(encode)
		if er != nil {
			return errs.ServerResponseFailed(er)
		}
	}
}

// Invoke -> server Invoke
func (s *Server) Invoke(ctx context.Context, req *message.Request) *message.Response {
	stub, ok := s.services[req.ServiceName]
	if !ok {
		return &message.Response{
			Version:    req.Version,
			Compresser: req.Compresser,
			Serializer: req.Serializer,
			MessageId:  req.MessageId,
			Error:      []byte(errs.InvalidServiceName.Error())}
	}
	return stub.Invoke(ctx, req)
}

// reflectionStub -> service stub
type reflectionStub struct {
	s           Service
	value       reflect.Value
	serializers []serialize.Serializer
}

// Invoke -> stub execute method by reflect
func (s *reflectionStub) Invoke(ctx context.Context, req *message.Request) *message.Response {
	method := s.value.MethodByName(req.Method)
	in := reflect.New(method.Type().In(1).Elem())
	//err := json.Unmarshal(reqData, in.Interface())
	response := &message.Response{
		Version:    req.Version,
		Compresser: req.Compresser,
		Serializer: req.Serializer,
		MessageId:  req.MessageId,
	}
	serializer := s.serializers[req.Serializer]
	err := serializer.Decode(req.Data, in.Interface())
	if err != nil {
		response.Error = []byte(err.Error())
		return response
	}
	res := method.Call(
		[]reflect.Value{reflect.ValueOf(ctx), in})

	if len(res) > 1 && res[1].Interface() != nil {
		response.Error = []byte(
			res[1].Interface().(error).Error())
		return response
	}
	data, err := serializer.Encode(res[0].Interface())
	if err != nil {
		response.Error = []byte(err.Error())
		return response
	}
	response.Data = data
	return response
}
