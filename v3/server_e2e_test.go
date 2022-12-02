//go:build e2e

package emicro

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestServer_Start(t *testing.T) {
	s := NewServer()
	s.RegisterService(&UserServiceServer{})
	err := s.Start(":8085")
	require.NoError(t, err)
}

type UserServiceServer struct {
}

func (u *UserServiceServer) ServiceName() string {
	return "user-service"
}

func (u *UserServiceServer) GetById(ctx context.Context, req *AnyRequest) (*AnyResponse, error) {
	return &AnyResponse{
		Msg: "Tom",
	}, nil
}
