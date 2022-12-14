package rpc

import (
	"context"
	"emicro/rpc/compress"
	"emicro/rpc/compress/gzip"
	"emicro/rpc/message"
	"emicro/rpc/serialize"
	"emicro/rpc/serialize/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_setFuncField(t *testing.T) {
	testSetFuncField(t, gzip.GzipCompressor{}, json.Serializer{})
}

func testSetFuncField(t *testing.T, compressor compress.Compressor, serializer serialize.Serializer) {
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
				t:          t,
				compressor: compressor,
				req: func() *message.Request {
					data := []byte(`{"msg":"123456"}`)
					req := &message.Request{
						MessageId:   2,
						Compresser:  compressor.Code(),
						ServiceName: "user-service",
						Method:      "GetById",
						Serializer:  serializer.Code(),
					}
					req.Data, _ = compressor.Compress(data)
					req.SetHeadLength()
					req.SetBodyLength()
					return req
				}(),
				resp: &message.Response{
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
			err := setFuncField(serializer, compressor, tc.service.s, tc.proxy)
			assert.Equal(t, tc.wantInitErr, err)
			if err != nil {
				return
			}
			resp, err := tc.service.do()
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.EqualValues(t, tc.wantResp, resp)
		})
	}
}

type mockProxy struct {
	t          *testing.T
	req        *message.Request
	resp       *message.Response
	compressor compress.Compressor
	serializer serialize.Serializer
	err        error
}

func (p *mockProxy) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	if p.err != nil {
		return nil, p.err
	}
	assert.Equal(p.t, p.req, req)
	var data []byte
	data, err := p.compressor.Compress(p.resp.Data)
	if err != nil {
		return nil, err
	}
	p.resp.Data = data
	return p.resp, nil
}

type mockService struct {
	s  Service
	do func() (any, error)
}

type UserServiceClient struct {
	GetById func(ctx context.Context, request *AnyRequest) (*AnyResponse, error)
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
