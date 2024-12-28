//go:build e2e

package ratelimit

import (
	"context"
	"emicro/proto/gen"
	"emicro/ratelimit"
	"errors"
	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func TestRedisFixWindowLimiter_BuildServerInterceptor(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	interceptor := ratelimit.NewRedisFixWindowLimiter(
		rdb, "user-service", 1, time.Second*3).BuildServerInterceptor()
	cnt := 0
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		cnt++
		return &gen.GetByIdResp{}, nil
	}
	resp, err := interceptor(context.Background(), &gen.GetByIdReq{}, &grpc.UnaryServerInfo{}, handler)
	require.NoError(t, err)
	assert.Equal(t, &gen.GetByIdResp{}, resp)

	resp, err = interceptor(context.Background(), &gen.GetByIdReq{}, &grpc.UnaryServerInfo{}, handler)
	require.Equal(t, errors.New("触及了瓶颈"), err)
	assert.Nil(t, resp)

	// 睡一个三秒，确保窗口新建了
	time.Sleep(time.Second * 3)
	resp, err = interceptor(context.Background(), &gen.GetByIdReq{}, &grpc.UnaryServerInfo{}, handler)
	require.NoError(t, err)
	assert.Equal(t, &gen.GetByIdResp{}, resp)
}
