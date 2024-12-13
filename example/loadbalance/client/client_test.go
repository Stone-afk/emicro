package client

import (
	"context"
	gen2 "emicro/example/proto/gen"
	v5 "emicro/v5"
	"emicro/v5/loadbalance"
	"emicro/v5/loadbalance/roundrobin"
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
	pickerBuilder := &roundrobin.WeightPickerBuilder{
		Filter: loadbalance.NewGroupFilterBuilder().Build(),
	}
	client := v5.NewClient(v5.ClientWithInsecure(),
		v5.ClientWithRegistry(r, time.Second*3),
		v5.ClientWithPickerBuilder(pickerBuilder.Name(), pickerBuilder))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// 设置目标分组
	ctx = context.WithValue(ctx, "group", "A")
	// 压力测试
	// ctx = context.WithValue(ctx, "group", "stress")
	// 设置初始化连接的 ctx
	conn, err := client.Dial(ctx, "user-service")
	cancel()
	require.NoError(t, err)
	userClient := gen2.NewUserServiceClient(conn)
	for i := 0; i < 10; i++ {
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
		resp, err := userClient.GetById(ctx, &gen2.GetByIdReq{
			Id: 12,
		})
		cancel()
		require.NoError(t, err)
		bs, err := json.Marshal(resp.User)
		require.NoError(t, err)
		t.Log(string(bs))
	}
}
