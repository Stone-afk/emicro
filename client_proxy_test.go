package emicro

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_setFuncField(t *testing.T) {
	testCases := []struct {
		name        string
		service     *mockService
		proxy       *mockProxy
		wantResp    any
		wantErr     error
		wantInitErr error
	}{
		{
			name: "proxy return error",
			service: func() *mockService {
				srv := &UserServiceClient{}
				return &mockService{
					s: srv,
					do: func() (any, error) {
						return srv.GetById(context.Background(), &AnyRequest{Msg: "123456"})
					},
				}
			}(),
			proxy: &mockProxy{
				t:   t,
				err: errors.New("mock error"),
			},
			wantErr: errors.New("mock error"),
		},
		{
			name: "user service",
			service: func() *mockService {
				srv := &UserServiceClient{}
				return &mockService{
					s: srv,
					do: func() (any, error) {
						return srv.GetById(context.Background(), &AnyRequest{Msg: "123456"})
					},
				}
			}(),
			proxy: &mockProxy{
				t: t,
				req: &Request{
					ServiceName: "user-service",
					Method:      "GetById",
					Data:        []byte(`{"msg":"123456"}`),
				},
				resp: &Response{
					Data: []byte(`{"msg":"这是123456的响应"}`),
				},
			},
			wantResp: &AnyResponse{
				Msg: "这是123456的响应",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := setFuncField(tc.service.s, tc.proxy)
			assert.Equal(t, tc.wantInitErr, err)
			if err != nil {
				return
			}
			resp, err := tc.service.do()
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantResp, resp)
		})
	}
}

// mockProxy
// 这里我们不用 mock 工具来生成，手写比较简单
type mockProxy struct {
	t    *testing.T
	req  *Request
	resp *Response
	err  error
}

func (p *mockProxy) Invoke(ctx context.Context, req *Request) (*Response, error) {
	if p.err != nil {
		return &Response{}, p.err
	}
	assert.Equal(p.t, p.req, req)
	return p.resp, nil
}

type mockService struct {
	s  service
	do func() (any, error)
}

type UserServiceClient struct {
	GetById func(cxt context.Context, req *AnyRequest) (*AnyResponse, error)
}

func (s *UserServiceClient) ServiceName() string {
	return "user-service"
}

type AnyRequest struct {
	Msg string `json:"msg"`
}

type AnyResponse struct {
	Msg string `json:"msg"`
}
