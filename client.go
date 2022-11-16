package emicro

import (
	"context"
	"emicro/internal/errs"
	"encoding/json"
	"fmt"
	"github.com/silenceper/pool"
	"net"
	"time"
)

type Client struct {
	connPool pool.Pool
}

type Proxy interface {
	Invoke(ctx context.Context, req *Request) (*Response, error)
}

func NewClient(address string) (*Client, error) {
	// Create a connection pool: Initialize the number of connections to 5,
	//the maximum idle connection is 20, and the maximum concurrent connection is 30
	poolConfig := &pool.Config{
		//连接池中拥有的最小连接数
		InitialCap: 5,
		//最大并发存活连接数
		MaxCap: 30,
		//最大空闲连接
		MaxIdle: 20,
		Factory: func() (interface{}, error) {
			return net.Dial("tcp", address)
		},
		Close: func(v interface{}) error {
			return v.(net.Conn).Close()
		},
		//连接最大空闲时间，超过该事件则将失效
		IdleTimeout: time.Minute,
	}
	connPool, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		return nil, err
	}
	client := &Client{
		connPool: connPool,
	}
	return client, nil
}

func (c *Client) Invoke(ctx context.Context, req *Request) (*Response, error) {
	conn, err := c.connPool.Get()
	if err != nil {
		return nil, fmt.Errorf("client: 无法获得获取一个可用连接 %w", err)
	}
	// put back
	defer func() {
		_ = c.connPool.Put(conn)
	}()
	cn := conn.(net.Conn)
	bs, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("client: 无法序列化请求, %w", err)
	}
	encode := EncodeMsg(bs)
	_, err = cn.Write(encode)
	if err != nil {
		return nil, err
	}
	bs, err = ReadMsg(cn)
	if err != nil {
		return nil, errs.ReadRespFailError
	}
	resp := &Response{}
	err = json.Unmarshal(bs, resp)
	return resp, err
}
