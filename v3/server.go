//go:build v3

package emicro

import (
	"context"
	"emicro/v3/internal/errs"
	"emicro/v3/serialize/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"time"
)

// Server -> tcp conn Server
type Server struct {
	listener    net.Listener
	services    map[string]*reflectionStub
	serializers []serialize.Serializer
}

// Close -> close net.Listener
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
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
	val := reflect.ValueOf(service)
	typ := reflect.TypeOf(service)
	methods := make(map[string]reflect.Value, val.NumMethod())
	for i := 0; i < val.NumMethod(); i++ {
		methodTyp := typ.Method(i)
		methods[methodTyp.Name] = val.Method(i)
	}
	s.services[service.ServiceName()] = &reflectionStub{
		s:           service,
		methods:     methods,
		serializers: s.serializers,
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
	s.listener = listener
	for {
		conn, err := listener.Accept()
		// 关闭了
		if err == net.ErrClosed {

			return nil
		}
		if err != nil {
			// 可以考虑打印日志
			fmt.Printf("server: accept connection got error: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

//// Start -> run server
//func (s *Server) Start(address string) error {
//	listener, err := net.Listen("tcp", address)
//	if err != nil {
//		return err
//	}
//	for {
//		conn, err := listener.Accept()
//		if err != nil {
//			// 可以考虑打印日志
//			fmt.Printf("server: accept connection got error: %v", err)
//		}
//		go func() {
//			if er := s.handleConn(conn); er != nil {
//				// 可以考虑打印日志
//				_ = conn.Close()
//				return
//			}
//		}()
//	}
//}

// handleConn -> handle tcp connection
func (s *Server) handleConn(conn net.Conn) {
	for {
		bs, err := ReadMsg(conn)
		if err == io.EOF {
			continue
		}
		if err != nil {
			return
		}

		req := message.DecodeReq(bs)
		ctx := context.Background()
		deadline, err := strconv.ParseInt(req.Meta["deadline"], 10, 64)
		cancel := func() {}
		if err == nil {
			ctx, cancel = context.WithDeadline(ctx, time.UnixMilli(deadline))
		}

		resp := s.Invoke(ctx, req)

		if req.Meta["one-way"] == "true" {
			// 什么也不需要处理。
			// 这样就相当于直接把连接资源释放了，去接收下一个请求了
			cancel()
			continue
		}

		// calculate and set the response head length
		resp.SetHeadLength()
		// calculate and set the response body length
		resp.SetBodyLength()
		encode := message.EncodeResp(resp)
		_, er := conn.Write(encode)
		if er != nil {
			fmt.Printf("server: sending response failed: %v", er)
		}
		cancel()
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
	serializers []serialize.Serializer
	methods     map[string]reflect.Value
}

// Invoke -> stub execute method by reflect
func (s *reflectionStub) Invoke(ctx context.Context, req *message.Request) *message.Response {
	response := &message.Response{
		Version:    req.Version,
		Compresser: req.Compresser,
		Serializer: req.Serializer,
		MessageId:  req.MessageId,
	}
	method, ok := s.methods[req.Method]
	if !ok {
		response.Error = []byte(
			errs.NotFoundServiceMethod(req.Method).Error())
		return response
	}
	in := reflect.New(method.Type().In(1).Elem())
	//err := json.Unmarshal(reqData, in.Interface())
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
