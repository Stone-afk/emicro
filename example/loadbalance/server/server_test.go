package server

import (
	v5 "emicro"
	"emicro/example/proto/gen"
	"emicro/registry/etcd"
	"fmt"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/sync/errgroup"
	"strconv"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	require.NoError(t, err)
	r, err := etcd.NewRegistry(etcdClient)
	require.NoError(t, err)
	var eg errgroup.Group
	for i := 0; i < 3; i++ {
		idx := i
		eg.Go(func() error {
			var group = "A"
			if idx%2 == 0 {
				group = "B"
				// 压力测试
				// group = "stress"
			}
			server := v5.NewServer("user-service",
				v5.ServerWithGroup(group),
				v5.ServerWithRegistry(r),
				v5.ServerWithTimeout(time.Second*3),
				v5.ServerWithWeight(uint32(1+idx)))
			require.NoError(t, err)
			defer func() {
				_ = server.Close()
			}()
			us := &UserServiceServer{
				group: group,
				name:  fmt.Sprintf("server-%d", idx),
			}
			gen.RegisterUserServiceServer(server, us)
			t.Log("启动服务器: " + us.name)
			// 端口分别是 8081, 8082, 8083
			return server.Start(":" + strconv.Itoa(8081+idx))
		})
	}
	err = eg.Wait()
	t.Log(err)
}
