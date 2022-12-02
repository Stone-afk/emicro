//go:build v2

package emicro

import (
	"context"
	"emicro/v2/message"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"testing"
)

func TestServer_handleConnection(t *testing.T) {
	// 用的是 json 来作为数据传输格式
	testCases := []struct {
		name     string
		conn     *mockConn
		service  Service
		wantResp []byte
	}{
		{
			name:    "user service",
			service: &UserService{},
			conn: &mockConn{
				readData: newRequestBytes(t, "user-service", "GetById", &AnyRequest{}),
			},
			wantResp: []byte(`{"msg":"这是GetById的响应"}`),
		},
	}
	for _, tc := range testCases {
		server := NewServer()
		err := server.RegisterService(tc.service)
		require.NoError(t, err)
		err = server.handleConn(tc.conn)
		require.NoError(t, err)
		resp := message.DecodeResp(tc.conn.writeData)
		assert.Equal(t, tc.wantResp, resp.Data)

	}
}

type mockConn struct {
	net.Conn
	readData  []byte
	readIndex int
	readErr   error

	writeData []byte
	writeErr  error
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

func newRequestBytes(t *testing.T, service string, method string, input any) []byte {
	data, err := json.Marshal(input)
	require.NoError(t, err)
	req := &message.Request{
		ServiceName: service,
		Method:      method,
		Data:        data,
		// 固定用 json
		Serializer: 1,
	}
	req.SetHeadLength()
	req.SetBodyLength()
	return message.EncodeReq(req)
}

func (m *mockConn) Read(bs []byte) (int, error) {
	if m.readIndex+len(bs) > len(m.readData) {
		// EOF是当没有更多输入可用时读取返回的错误
		return 0, io.EOF
	}
	// copy(bs, m.readData[m.readIndex:])
	for i := 0; i < len(bs); i++ {
		bs[i] = m.readData[m.readIndex+i]
	}
	m.readIndex = m.readIndex + len(bs)
	return len(bs), m.readErr
}

func (m *mockConn) Write(bs []byte) (int, error) {
	m.writeData = append(m.writeData, bs...)
	return len(bs), m.writeErr
}
