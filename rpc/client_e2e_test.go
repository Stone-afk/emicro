//go:build e2e

package rpc

import (
	"context"
	"emicro/rpc/compress/gzip"
	"emicro/rpc/serialize/json"
	"github.com/stretchr/testify/require"
	"log"
	"testing"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient(":8085")
	require.NoError(t, err)
	us := &UserServiceClient{}
	serializer := json.Serializer{}
	compressor := gzip.Compressor{}
	err = setFuncField(serializer, compressor, us, c)
	require.NoError(t, err)

	resp, err := us.GetById(context.Background(), &AnyRequest{
		Msg: "100",
	})
	require.NoError(t, err)
	log.Println(resp)
}
