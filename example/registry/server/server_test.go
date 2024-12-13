package server

import (
	"context"
	gen2 "emicro/example/proto/gen"
	v5 "emicro/v5"
	"emicro/v5/registry/etcd"
	"fmt"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"testing"
)

func TestServer(t *testing.T) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	require.NoError(t, err)
	r, err := etcd.NewRegistry(etcdClient)
	require.NoError(t, err)
	us := &UserServiceServer{}
	server := v5.NewServer("user-service", v5.ServerWithRegistry(r))
	require.NoError(t, err)
	gen2.RegisterUserServiceServer(server, us)
	t.Log("开始启动服务器")
	// 我在这里调用 Start 方法，就意味着 us 已经完全准备好了
	if err = server.Start(":8081"); err != nil {
		t.Log(err)
	}
}

type UserServiceServer struct {
	gen2.UnimplementedUserServiceServer
}

func (u *UserServiceServer) GetById(ctx context.Context, req *gen2.GetByIdReq) (*gen2.GetByIdResp, error) {
	fmt.Printf("user id: %d", req.Id)
	return &gen2.GetByIdResp{
		User: &gen2.User{
			Id:     req.Id,
			Status: 123,
		},
	}, nil
}
