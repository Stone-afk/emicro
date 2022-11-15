package emicro

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_setFuncField(t *testing.T) {
	testCases := []struct {
		name     string
		service  *mockService
		proxy    *mockProxy
		wantResp any
		wantErr  error
	}{
		{},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := setFuncField(tc.service.s, tc.proxy)
			assert.Equal(t, tc.wantErr, err)
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
}

func (p *mockProxy) Invoke(ctx context.Context, req *Request) (*Response, error) {
	return &Response{}, nil
}

type mockService struct {
	s  service
	do func() (any, error)
}

type UserServiceClient struct {
	GetById func(cxt context.Context, req *AnyRequest) (*AnyResponse, error)
}

type AnyRequest struct {
	Msg string `json:"msg"`
}

type AnyResponse struct {
	Msg string `json:"msg"`
}
