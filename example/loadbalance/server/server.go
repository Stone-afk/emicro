package main

import (
	"emicro"
	"emicro/example/proto/gen"
	"emicro/registry/etcd"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/sync/errgroup"
	"strconv"
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
	var eg errgroup.Group
	for i := 0; i < 3; i++ {
		idx := i
		eg.Go(func() error {
			group := "a"
			if idx%2 == 0 {
				group = "b"
			}
			server := emicro.NewServer("user-service",
				emicro.ServerWithGroup(group),
				emicro.ServerWithRegistry(r),
				emicro.ServerWithTimeout(time.Second*3),
				emicro.ServerWithWeight(uint32(1+idx)))
			defer func() {
				_ = server.Close()
			}()
			us := &UserService{
				name: fmt.Sprintf("server-%d", idx),
			}
			gen.RegisterUserServiceServer(server, us)
			fmt.Println("启动服务器: " + us.name)
			// 端口分别是 8081, 8082, 8083
			return server.Start(":" + strconv.Itoa(8081+idx))
		})
	}
	err = eg.Wait()
	fmt.Println(err)
}
