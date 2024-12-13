package server

import (
	"context"
	gen2 "emicro/example/proto/gen"
	"fmt"
)

type UserServiceServer struct {
	name  string
	group string
	gen2.UnimplementedUserServiceServer
}

func (u *UserServiceServer) GetById(ctx context.Context, req *gen2.GetByIdReq) (*gen2.GetByIdResp, error) {
	//go func() {
	// 转异步
	//	fmt.Println(s.group)
	//	// 做一些事情
	//}()
	// 返回一个 202
	fmt.Printf("server %s, group %s, get user id: %d \n", u.name, u.group, req.Id)
	return &gen2.GetByIdResp{
		User: &gen2.User{
			Id:     req.Id,
			Status: 123,
		},
	}, nil
}
