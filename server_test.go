package emicro

import (
	"context"
	"testing"
)

func TestServer_handleConnection(t *testing.T) {

}

type UserService struct {
}

func (u *UserService) ServiceName() string {
	return "user-service"
}

func (u *UserService) GetById(ctx context.Context, request *AnyRequest) (*AnyResponse, error) {
	return &AnyResponse{
		Msg: "这是GetById的响应",
	}, nil
}