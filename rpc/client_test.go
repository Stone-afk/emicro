package rpc

import (
	"context"
	"emicro/proto/gen"
	"emicro/rpc/compress/gzip"
	message2 "emicro/rpc/message"
	"emicro/rpc/serialize/json"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func Test_setFuncField(t *testing.T) {
	testCases := []struct {
		name string

		mock func(ctrl *gomock.Controller) Proxy

		service Service
		wantErr error
	}{
		{
			name:    "nil",
			service: nil,
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("rpc: 不支持 nil"),
		},
		{
			name:    "no pointer",
			service: &UserServiceClient{},
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("rpc: 只支持指向结构体的一级指针"),
		},
		{
			name: "user service",
			mock: func(ctrl *gomock.Controller) Proxy {
				p := NewMockProxy(ctrl)
				p.EXPECT().Invoke(gomock.Any(), &message2.Request{
					ServiceName: "user-service",
					MethodName:  "GetById",
					Data:        []byte(`{"Id":123}`),
				}).Return(&message2.Response{}, nil)
				return p
			},
			service: &UserService{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			err := setFuncField(json.Serializer{}, gzip.Compressor{}, tc.service, tc.mock(ctrl))
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			resp, err := tc.service.(*UserService).GetById(context.Background(), &GetByIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, err)
			t.Log(resp)
		})
	}
}

type UserServiceClient struct {
	// 用反射来赋值
	// 类型是函数的字段，它不是方法（它不是定义在 UserService 上的方法）
	// 本质上是一个字段
	GetById func(ctx context.Context, req *GetByIdReq) (*GetByIdResp, error)

	GetByIdProto func(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error)
}

func (s *UserServiceClient) Name() string {
	return "user-service"
}

type GetByIdReq struct {
	Id int
}

type GetByIdResp struct {
	Msg string
}

type UserServiceServer struct {
	Err error
	Msg string
}

func (u *UserServiceServer) GetById(ctx context.Context, req *GetByIdReq) (*GetByIdResp, error) {
	log.Println(req)
	return &GetByIdResp{
		Msg: u.Msg,
	}, u.Err
}

func (u *UserServiceServer) GetByIdProto(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error) {
	log.Println(req)
	return &gen.GetByIdResp{
		User: &gen.User{
			Name: u.Msg,
		},
	}, u.Err
}

func (u *UserServiceServer) Name() string {
	return "user-service"
}

type UserServiceServerTimeout struct {
	t     *testing.T
	sleep time.Duration
	Err   error
	Msg   string
}

func (u *UserServiceServerTimeout) GetById(ctx context.Context, req *GetByIdReq) (*GetByIdResp, error) {
	if _, ok := ctx.Deadline(); !ok {
		u.t.Fatal("没有设置超时")
	}
	time.Sleep(u.sleep)
	return &GetByIdResp{
		Msg: u.Msg,
	}, u.Err
}

func (u *UserServiceServerTimeout) Name() string {
	return "user-service"
}
