package rpc

import (
	"context"
	"emicro/internal/errs"
	"emicro/rpc/compress"
	message2 "emicro/rpc/message"
	"emicro/rpc/serialize"
	"emicro/rpc/serialize/json"
	"emicro/rpc/tcp"
	"errors"
	"github.com/gotomicro/ekit/bean/option"
	"github.com/silenceper/pool"
	"net"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"
)

var _ Proxy = (*Client)(nil)

type ClientOption func(client *Client)

// messageId
var messageId uint32 = 0

// Client -> tcp conn client
type Client struct {
	connPool   pool.Pool
	serializer serialize.Serializer
	compressor compress.Compressor
}

// InitClientProxy -> init client proxy
func InitClientProxy(address string, srv Service) error {
	client, err := NewClient(address)
	if err != nil {
		return err
	}
	if err = setFuncField(client.serializer, client.compressor, srv, client); err != nil {
		return err
	}
	return nil
}

// InitService -> init client proxy
func (c *Client) InitService(srv Service) error {
	return setFuncField(c.serializer, c.compressor, srv, c)
}

// setFuncField
func setFuncField(serializer serialize.Serializer,
	compress compress.Compressor, service Service, proxy Proxy) error {
	srvValElem := reflect.ValueOf(service).Elem()
	srvTypElem := srvValElem.Type()
	if srvTypElem.Kind() == reflect.Ptr {
		return errs.ServiceTypError
	}
	numField := srvTypElem.NumField()
	for i := 0; i < numField; i++ {
		//fieldTyp := srvTypElem.Field(i).Type
		structField := srvTypElem.Field(i)
		fieldVal := srvValElem.Field(i)

		if !fieldVal.CanSet() {
			continue
		}
		fn := func(args []reflect.Value) (results []reflect.Value) {
			in := args[1].Interface()
			// out := reflect.New(fieldTyp.Type.Out(0).Elem()).Interface()
			// out := reflect.Zero(structField.Type.Out(0))
			out := reflect.New(structField.Type.Out(0).Elem())
			// serialize request data
			reqData, err := serializer.Encode(in)
			if err != nil {
				return []reflect.Value{out, reflect.ValueOf(err)}
			}
			// compress response data
			reqData, err = compress.Compress(reqData)
			if err != nil {
				return []reflect.Value{out, reflect.ValueOf(err)}
			}
			ctx := args[0].Interface().(context.Context)
			// For the time being, write it dead first.
			//Later, we will consider the general link metadata transmission and reconstruction
			var meta map[string]string
			if isOneway(ctx) {
				meta = map[string]string{"one-way": "true"}
			}
			if deadline, ok := ctx.Deadline(); ok {
				// More space is required for string transmission
				meta["deadline"] = strconv.FormatInt(deadline.UnixMilli(), 10)
			}
			req := &message2.Request{
				Meta:        meta,
				Compresser:  compress.Code(),
				Serializer:  serializer.Code(),
				ServiceName: service.Name(),
				MethodName:  structField.Name,
				MessageId:   atomic.AddUint32(&messageId, +1),
			}
			// calculate and set the request head length
			req.CalculateHeaderLength()
			// calculate and set the request body length
			req.CalculateBodyLength()
			resp, err := proxy.Invoke(ctx, req)
			if err != nil {
				return []reflect.Value{out, reflect.ValueOf(err)}
			}
			var respErr error
			if len(resp.Error) > 0 {
				respErr = errors.New(string(resp.Error))
			}
			if len(resp.Data) > 0 {
				//out := reflect.Zero(structField.Type.Out(0))
				var data []byte
				// decompress response data
				data, err = compress.UnCompress(resp.Data)
				if err != nil {
					return []reflect.Value{out, reflect.ValueOf(err)}
				}
				// deserialize response data
				err = serializer.Decode(data, out.Interface())
				if err != nil {
					return []reflect.Value{out, reflect.ValueOf(err)}
				}
			}
			var errVal reflect.Value
			if respErr == nil {

			} else {
				errVal = reflect.ValueOf(respErr)
			}
			return []reflect.Value{out, errVal}
		}
		fieldVal.Set(reflect.MakeFunc(structField.Type, fn))
	}
	return nil
}

// Invoke -> invoke rpc service
func (c *Client) Invoke(ctx context.Context, request *message2.Request) (*message2.Response, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var (
		resp *message2.Response
		err  error
	)
	ch := make(chan struct{})
	go func() {
		encode := message2.EncodeReq(request)
		resp, err = c.doInvoke(ctx, encode)
		ch <- struct{}{}
		close(ch)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		return resp, err
	}
}

// doInvoke -> invoke rpc service
func (c *Client) doInvoke(ctx context.Context, encode []byte) (*message2.Response, error) {
	val, err := c.connPool.Get()
	if err != nil {
		return nil, errs.ClientConnDeaded(err)
	}
	// put back
	defer func() {
		_ = c.connPool.Put(val)
	}()
	conn := val.(net.Conn)
	l, err := conn.Write(encode)
	if err != nil {
		return nil, err
	}
	if l != len(encode) {
		return nil, errs.ClientNotAllWritten
	}
	if isOneway(ctx) {
		return nil, errs.OnewayError
	}
	data, err := tcp.ReadMsg(conn)
	if err != nil {
		return nil, errs.ReadRespFailError
	}
	return message2.DecodeResp(data), nil
}

// ClientWithSerializer -> option
func ClientWithSerializer(s serialize.Serializer) option.Option[Client] {
	return func(client *Client) {
		client.serializer = s
	}
}

// ClientWithCompressor -> option
func ClientWithCompressor(c compress.Compressor) option.Option[Client] {
	return func(client *Client) {
		client.compressor = c
	}
}

// NewClient -> create Client
func NewClient(address string, opts ...option.Option[Client]) (*Client, error) {
	poolConfig := &pool.Config{
		InitialCap: 5,
		MaxIdle:    20,
		MaxCap:     30,
		Factory: func() (interface{}, error) {
			return net.Dial("tcp", address)
		},
		Close: func(i interface{}) error {
			return i.(net.Conn).Close()
		},
		IdleTimeout: time.Minute,
	}
	connPool, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		return nil, err
	}
	client := &Client{
		connPool:   connPool,
		serializer: json.Serializer{},
		// 避免 nil 检测
		compressor: compress.DoNothingCompressor{},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client, nil
}
