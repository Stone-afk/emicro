package main

import (
	"context"
	v5 "emicro/v5"
	"emicro/v5/example/proto/gen"
	"emicro/v5/loadbalance"
	"emicro/v5/loadbalance/roundrobin"
	"emicro/v5/registry/etcd"
	"encoding/json"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func main() {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	if err != nil {
		panic(err)
	}
	r, err := etcd.NewRegistry(etcdClient)
	if err != nil {
		panic(err)
	}
	pickerBuilder := &roundrobin.WeightPickerBuilder{
		Filter: loadbalance.NewGroupFilterBuilder().Build(),
	}
	client := v5.NewClient(v5.ClientInsecure(),
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
	if err != nil {
		panic(err)
	}
	userClient := gen.NewUserServiceClient(conn)
	for i := 0; i < 10; i++ {
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
		resp, err := userClient.GetById(ctx, &gen.GetByIdReq{
			Id: 12,
		})
		cancel()
		if err != nil {
			panic(err)
		}
		bs, err := json.Marshal(resp.User)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(bs))
	}
}
