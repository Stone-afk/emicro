package emicro

import (
	"context"
	"emicro/internal/errs"
	"emicro/message"
	"encoding/json"
	"github.com/silenceper/pool"
	"net"
	"reflect"
	"time"
)

// Client -> tcp conn client
type Client struct {
	connPool pool.Pool
}

// Invoke -> invoke rpc service
func (c *Client) Invoke(ctx context.Context, request *message.Request) (*message.Response, error) {
	return &message.Response{}, nil
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

// InitProxyClient -> init client proxy
func InitProxyClient(address string, srv Service) error {
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
				ServiceName: s.ServiceName(),
				Method:      fieldTyp.Name,
				Data:        reqData,
			}
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
