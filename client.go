package emicro

import "context"

type Client struct {
}

type Proxy interface {
	Invoke(ctx context.Context, req *Request) (*Response, error)
}

func NewClient(address string) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Invoke(ctx context.Context, req *Request) (*Response, error) {
	return &Response{}, nil
}
