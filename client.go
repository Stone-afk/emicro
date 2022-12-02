package emicro

import (
	"context"
	"emicro/internal/errs"
	"emicro/message"
	"emicro/serialize"
	"emicro/serialize/json"
	"errors"
	"github.com/gotomicro/ekit/bean/option"
	"github.com/silenceper/pool"
	"net"
	"reflect"
	"sync/atomic"
	"time"
)

// messageId
var messageId uint32 = 0

// Client -> tcp conn client
type Client struct {
	connPool   pool.Pool
	serializer serialize.Serializer
}

// ClientWithSerializer -> option
func ClientWithSerializer(s serialize.Serializer) option.Option[Client] {
	return func(client *Client) {
		client.serializer = s
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
	}
	for _, opt := range opts {
		opt(client)
	}
	return client, nil

}

// InitClientProxy -> init client proxy
func InitClientProxy(address string, srv Service) error {
	client, err := NewClient(address)
	if err != nil {
		return err
	}
	if err = setFuncField(client.serializer, srv, client); err != nil {
		return err
	}
	return nil
}

// InitClientProxy -> init client proxy
func (c *Client) InitClientProxy(srv Service) error {
	return setFuncField(c.serializer, srv, c)
}

// setFuncField
func setFuncField(serializer serialize.Serializer, service Service, proxy Proxy) error {
	srvValElem := reflect.ValueOf(service).Elem()
	srvTypElem := srvValElem.Type()
	if srvTypElem.Kind() == reflect.Ptr {
		return errs.ServiceTypError
	}
	numField := srvTypElem.NumField()
	for i := 0; i < numField; i++ {
		fieldTyp := srvTypElem.Field(i)
		fieldVal := srvValElem.Field(i)

		if !fieldVal.CanSet() {
			continue
		}
		fn := func(args []reflect.Value) (results []reflect.Value) {
			ctx := args[0].Interface().(context.Context)
			in := args[1].Interface()
			// out := reflect.New(fieldTyp.Type.Out(0).Elem()).Interface()
			out := reflect.Zero(fieldTyp.Type.Out(0))
			reqData, err := serializer.Encode(in)
			if err != nil {
				return []reflect.Value{out, reflect.ValueOf(err)}
			}
			req := &message.Request{
				MessageId:   atomic.AddUint32(&messageId, +1),
				Compresser:  0,
				Serializer:  serializer.Code(),
				ServiceName: service.ServiceName(),
				Method:      fieldTyp.Name,
				Data:        reqData,
			}
			// calculate and set the request head length
			req.SetHeadLength()
			// calculate and set the request body length
			req.SetBodyLength()
			resp, err := proxy.Invoke(ctx, req)
			if err != nil {
				return []reflect.Value{out, reflect.ValueOf(err)}
			}
			var respErr error
			if len(resp.Error) > 0 {
				respErr = errors.New(string(resp.Error))
			}
			if len(resp.Data) > 0 {
				out = reflect.New(fieldTyp.Type.Out(0).Elem())
				err = serializer.Decode(resp.Data, out.Interface())
				if err != nil {
					return []reflect.Value{out, reflect.ValueOf(err)}
				}
			}
			var errVal reflect.Value
			if respErr == nil {
				errVal = reflect.Zero(reflect.TypeOf(new(error)).Elem())
			} else {
				errVal = reflect.ValueOf(respErr)
			}
			return []reflect.Value{out, errVal}
		}
		fieldVal.Set(reflect.MakeFunc(fieldTyp.Type, fn))
	}
	return nil
}

// Invoke -> invoke rpc service
func (c *Client) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	val, err := c.connPool.Get()
	if err != nil {
		return nil, errs.ClientConnDeaded(err)
	}
	// put back
	defer func() {
		_ = c.connPool.Put(val)
	}()
	conn := val.(net.Conn)
	encode := message.EncodeReq(req)
	l, err := conn.Write(encode)
	if err != nil {
		return nil, err
	}
	if l != len(encode) {
		return nil, errs.ClientNotAllWritten
	}
	data, err := ReadMsg(conn)
	if err != nil {
		return nil, errs.ReadRespFailError
	}

	return message.DecodeResp(data), nil
}
