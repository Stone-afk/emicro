package main

import (
	"context"
	"emicro"
	"emicro/example/proto/gen"
	"emicro/loadbalance"
	"emicro/loadbalance/roundrobin"
	"emicro/registry/etcd"
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
		Filter: loadbalance.GroupFilter,
	}
	client := emicro.NewClient(emicro.ClientWithInsecure(),
		emicro.ClientWithRegistry(r, time.Second*3),
		emicro.ClientWithPickerBuilder(pickerBuilder.Name(), pickerBuilder))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
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
