package rpc

import (
	"context"
	"emicro/internal/errs"
	"emicro/v5/rpc/compress"
	"emicro/v5/rpc/message"
	"emicro/v5/rpc/serialize"
	"emicro/v5/rpc/serialize/json"
	"emicro/v5/rpc/tcp"
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
	compressors []compress.Compressor
}

// Close -> close net.Listener
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
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

// Start -> run server
func (s *Server) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	s.listener = listener
	for {
		conn, err := listener.Accept()
		// closed
		if err == net.ErrClosed {
			return nil
		}
		if err != nil {
			// consider printing logs
			fmt.Printf("server: accept connection got error: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

// handleConn -> handle tcp connection
func (s *Server) handleConn(conn net.Conn) {
	for {
		bs, err := tcp.ReadMsg(conn)
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
			// nothing needs to be dealt with.
			// this is equivalent to directly releasing the connection resources to receive the next request
			cancel()
			continue
		}
		// calculate and set the response head length
		resp.CalculateHeaderLength()
		// calculate and set the response body length
		resp.CalculateBodyLength()
		encode := message.EncodeResp(resp)
		_, err = conn.Write(encode)
		if err != nil {
			fmt.Printf("server: sending response failed: %v", err)
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
			Error:      []byte(errs.InvalidServiceName.Error()),
		}
	}
	return stub.Invoke(ctx, req)
}

// NewServer instance
func NewServer() *Server {
	res := &Server{
		services: make(map[string]*reflectionStub, 8),
		// A byte can have up to 256 implementations, which can be directly made into a simple bit array
		// 一个字节，最多有 256 个实现，直接做成一个简单的 bit array 的东西
		serializers: make([]serialize.Serializer, 256),
		compressors: make([]compress.Compressor, 256),
	}
	// Register the most basic serialization protocol
	res.RegisterSerializer(json.Serializer{})
	res.RegisterCompressor(compress.DoNothingCompressor{})
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
	s.services[service.Name()] = &reflectionStub{
		s:           service,
		methods:     methods,
		serializers: s.serializers,
		compressors: s.compressors,
	}
	return nil
}

// RegisterSerializer -> register serializer
func (s *Server) RegisterSerializer(serializer serialize.Serializer) {
	s.serializers[serializer.Code()] = serializer
}

// RegisterCompressor -> register compressor
func (s *Server) RegisterCompressor(compressor compress.Compressor) {
	s.compressors[compressor.Code()] = compressor
}

// reflectionStub -> service stub
type reflectionStub struct {
	s           Service
	serializers []serialize.Serializer
	compressors []compress.Compressor
	methods     map[string]reflect.Value
}

// Invoke -> stub execute method by reflect
func (s *reflectionStub) Invoke(ctx context.Context, req *message.Request) *message.Response {
	response := &message.Response{
		Version:    req.Version,
		Compresser: req.Compresser,
		// Theoretically, you can use another serialization protocol here,
		// but it is unnecessary to expose this function to users
		Serializer: req.Serializer,
		MessageId:  req.MessageId,
	}
	method, ok := s.methods[req.MethodName]
	if !ok {
		response.Error = []byte(errs.NotFoundServiceMethod(req.MethodName).Error())
		return response
	}
	in := reflect.New(method.Type().In(1).Elem())

	// decompress request data
	compresser := s.compressors[req.Compresser]
	reqData, err := compresser.UnCompress(req.Data)
	if err != nil {
		response.Error = []byte(err.Error())
		return response
	}
	// deserialize Request Data
	// err := json.Unmarshal(reqData, in.Interface())
	serializer := s.serializers[req.Serializer]
	err = serializer.Decode(reqData, in.Interface())
	if err != nil {
		response.Error = []byte(err.Error())
		return response
	}
	res := method.Call([]reflect.Value{reflect.ValueOf(ctx), in})
	if len(res) > 1 && res[1].Interface() != nil {
		response.Error = []byte(res[1].Interface().(error).Error())
		return response
	}
	// serialize response data
	respData, err := serializer.Encode(res[0].Interface())
	if err != nil {
		// server error
		response.Error = []byte(err.Error())
		return response
	}
	// compress response data
	respData, err = compresser.Compress(respData)
	if err != nil {
		response.Error = []byte(err.Error())
		return response
	}
	response.Data = respData
	return response
}
