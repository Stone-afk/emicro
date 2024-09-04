package client

import (
	"context"
	v5 "emicro/v5"
	"emicro/v5/example/proto/gen"
	"emicro/v5/registry/etcd"
	"encoding/json"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	require.NoError(t, err)
	r, err := etcd.NewRegistry(etcdClient)
	require.NoError(t, err)
	client := v5.NewClient(v5.ClientWithInsecure(), v5.ClientWithRegistry(r, time.Second*3))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// 设置初始化连接的 ctx
	conn, err := client.Dial(ctx, "user-service")
	cancel()
	require.NoError(t, err)
	userClient := gen.NewUserServiceClient(conn)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	resp, err := userClient.GetById(ctx, &gen.GetByIdReq{
		Id: 12,
	})
	cancel()
	require.NoError(t, err)
	bs, err := json.Marshal(resp.User)
	require.NoError(t, err)
	t.Log(string(bs))
}
