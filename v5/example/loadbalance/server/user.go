package server

import (
	"context"
	"emicro/v5/example/proto/gen"
	"fmt"
)

type UserServiceServer struct {
	name  string
	group string
	gen.UnimplementedUserServiceServer
}

func (u *UserServiceServer) GetById(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error) {
	//go func() {
	// 转异步
	//	fmt.Println(s.group)
	//	// 做一些事情
	//}()
	// 返回一个 202
	fmt.Printf("server %s, group %s, get user id: %d \n", u.name, u.group, req.Id)
	return &gen.GetByIdResp{
		User: &gen.User{
			Id:     req.Id,
			Status: 123,
		},
	}, nil
}
