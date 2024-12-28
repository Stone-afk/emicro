package broadcast

import (
	"context"
	"emicro"
	gen2 "emicro/proto/gen"
	"emicro/registry/etcd"
	"fmt"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func TestUseBroadCast(t *testing.T) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	require.NoError(t, err)
	r, err := etcd.NewRegistry(etcdClient)
	require.NoError(t, err)

	var eg errgroup.Group
	var servers []*UserServiceServer
	for i := 0; i < 3; i++ {
		server := emicro.NewServer("user-service", emicro.ServerWithRegistry(r))
		us := &UserServiceServer{
			idx: i,
		}
		servers = append(servers, us)
		gen2.RegisterUserServiceServer(server, us)
		// 启动 8081,8082, 8083 三个端口
		port := fmt.Sprintf(":808%d", i+1)
		eg.Go(func() error {
			return server.Start(port)
		})
		defer func() {
			_ = server.Close()
		}()
	}

	client := emicro.NewClient(emicro.ClientWithInsecure(),
		emicro.ClientWithRegistry(r, time.Second*3))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	require.NoError(t, err)
	ctx = UsingBroadCast(ctx)
	bd := NewClusterBuilder(r, "user-service", grpc.WithInsecure())
	cc, err := client.Dial(ctx, "user-service", grpc.WithUnaryInterceptor(bd.BuildUnaryClientInterceptor()))
	require.NoError(t, err)
	uc := gen2.NewUserServiceClient(cc)
	resp, err := uc.GetById(ctx, &gen2.GetByIdReq{Id: 13})
	require.NoError(t, err)
	t.Log(resp)
	for _, s := range servers {
		require.Equal(t, 1, s.cnt)
	}

}

type UserServiceServer struct {
	idx int
	cnt int
	gen2.UnimplementedUserServiceServer
}

func (s *UserServiceServer) GetById(ctx context.Context, req *gen2.GetByIdReq) (*gen2.GetByIdResp, error) {
	s.cnt++
	return &gen2.GetByIdResp{
		User: &gen2.User{
			Name: fmt.Sprintf("hello, world %d", s.idx),
		},
	}, nil
}
