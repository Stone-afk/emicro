package emicro

import (
	"context"
	"emicro/internal/errs"
	"emicro/message"
	"encoding/json"
	"errors"
	"fmt"
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
	connPool pool.Pool
}

// NewClient -> create Client
func NewClient(address string) (*Client, error) {
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
	return &Client{connPool: connPool}, nil

}

// InitClientProxy -> init client proxy
func InitClientProxy(address string, srv Service) error {
	client, err := NewClient(address)
	if err != nil {
		return err
	}
	if err = setFuncField(srv, client); err != nil {
		return err
	}
	return nil
}

// setFuncField
func setFuncField(s Service, p Proxy) error {
	srvValElem := reflect.ValueOf(s).Elem()
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
			out := reflect.New(fieldTyp.Type.Out(0).Elem()).Interface()
			reqData, err := json.Marshal(in)
			if err != nil {
				return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
			}
			req := &message.Request{
				MessageId:   atomic.AddUint32(&messageId, +1),
				Compresser:  0,
				Serializer:  0,
				ServiceName: s.ServiceName(),
				Method:      fieldTyp.Name,
				Data:        reqData,
			}
			// calculate and set the request head length
			req.SetHeadLength()
			// calculate and set the request body length
			req.SetBodyLength()
			resp, err := p.Invoke(ctx, req)
			if err != nil {
				return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
			}
			err = json.Unmarshal(resp.Data, out)
			if err != nil {
				return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
			}
			// nilErr := reflect.Zero(reflect.TypeOf(new(error)).Elem())
			return []reflect.Value{reflect.ValueOf(out), reflect.Zero(reflect.TypeOf(new(error)).Elem())}
		}
		fieldVal.Set(reflect.MakeFunc(fieldTyp.Type, fn))
	}
	return nil

}

// Invoke -> invoke rpc service
func (c *Client) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	val, err := c.connPool.Get()
	if err != nil {
		return nil, fmt.Errorf("client: unable to get an available connection %w", err)
	}
	// put back
	defer func() {
		_ = c.connPool.Put(val)
	}()
	conn := val.(net.Conn)
	//reqBs, err := json.Marshal(req)
	//if err != nil {
	//	return nil, fmt.Errorf("client: unable to serialize request, %w", err)
	//}
	//encode := EncodeMsg(reqBs)
	encode := message.EncodeReq(req)
	l, err := conn.Write(encode)
	if err != nil {
		return nil, err
	}
	if l != len(encode) {
		return nil, errors.New("micro: 未写入全部数据")
	}
	data, err := ReadMsg(conn)
	if err != nil {
		return nil, errs.ReadRespFailError
	}
	// resp := &message.Response{}
	//err = json.Unmarshal(data, resp)
	//if err != nil {
	//	return nil, fmt.Errorf("client: unable to deserialize response, %w", err)
	//}
	return message.DecodeResp(data), nil
}
