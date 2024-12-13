package main

import (
	"context"
	"emicro/example/proto/gen"
	"errors"
	"time"
)

type UserService struct {
}

func (u *UserService) GetById(ctx context.Context, req *FindByUserIdReq) (*FindByUserIdResp, error) {
	return &FindByUserIdResp{
		User: &User{
			Id:         12,
			Name:       "Tom",
			Avatar:     "http://my-avatar",
			Email:      "xxx@xxx.com",
			Password:   "123456",
			CreateTime: time.Now().Second(),
		},
	}, nil
}

func (u *UserService) AlwaysError(ctx context.Context, req *FindByUserIdReq) (*FindByUserIdResp, error) {
	return nil, errors.New("this is an error")
}

func (u *UserService) Name() string {
	return "user"
}

// UserServiceProto 用来测试 protobuf 协议
type UserServiceProto struct {
}

func (u *UserServiceProto) GetById(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error) {
	return &gen.GetByIdResp{
		User: &gen.User{
			Id: 123,
		},
	}, nil
}

func (u *UserServiceProto) Name() string {
	return "user-service-proto"
}
