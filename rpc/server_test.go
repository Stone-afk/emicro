package rpc

import (
	"context"
	"emicro/rpc/compress"
	"emicro/rpc/compress/gzip"
	message2 "emicro/rpc/message"
	"emicro/rpc/tcp"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestServer_handleConnection(t *testing.T) {
	testServerHandleConnection(t, gzip.Compressor{})
}

func testServerHandleConnection(t *testing.T, c compress.Compressor) {
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
				readData: newRequestBytes(t, "user-service", "GetById", &AnyRequest{}, c),
			},
			wantResp: func() []byte {
				data := []byte(`{"msg":"这是GetById的响应"}`)
				data, _ = c.Compress(data)
				return data
			}(),
		},
	}
	for _, tc := range testCases {
		server := NewServer()
		err := server.RegisterService(tc.service)
		server.RegisterCompressor(c)
		require.NoError(t, err)
		err = server.TestHandleConn(tc.conn)
		require.NoError(t, err)
		resp := message2.DecodeResp(tc.conn.writeData)
		assert.Equal(t, tc.wantResp, resp.Data)

	}
}

// handleConn -> handle tcp connection
func (s *Server) TestHandleConn(conn net.Conn) error {
	for {
		bs, err := tcp.ReadMsg(conn)
		if err == io.EOF {
			continue
		}
		if err != nil {
			return fmt.Errorf("emicro: server sending response failed: %v", err)
		}

		req := message2.DecodeReq(bs)
		ctx := context.Background()
		deadline, err := strconv.ParseInt(req.Meta["deadline"], 10, 64)
		cancel := func() {}
		if err == nil {
			ctx, cancel = context.WithDeadline(ctx, time.UnixMilli(deadline))
		}

		resp := s.Invoke(ctx, req)

		if req.Meta["one-way"] == "true" {
			// 什么也不需要处理。
			// 这样就相当于直接把连接资源释放了，去接收下一个请求了
			cancel()
			continue
		}

		// calculate and set the response head length
		resp.CalculateHeaderLength()
		// calculate and set the response body length
		resp.CalculateBodyLength()
		encode := message2.EncodeResp(resp)
		_, er := conn.Write(encode)
		if er != nil {
			return fmt.Errorf("emicro: server sending response failed: %v", er)
		}
		cancel()
		return nil
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

type UserService struct{}

func (u *UserService) Name() string {
	return "user-service"
}

func (u *UserService) GetById(ctx context.Context, request *AnyRequest) (*AnyResponse, error) {
	return &AnyResponse{
		Msg: "这是GetById的响应",
	}, nil
}

type AnyRequest struct {
	Msg string `json:"msg"`
}

type AnyResponse struct {
	Msg string `json:"msg"`
}

func newRequestBytes(t *testing.T, service string, method string, input any, c compress.Compressor) []byte {
	data, err := json.Marshal(input)
	require.NoError(t, err)
	data, err = c.Compress(data)
	require.NoError(t, err)
	req := &message2.Request{
		ServiceName: service,
		MethodName:  method,
		Data:        data,
		// 固定用 json
		Serializer: 1,
		Compresser: 1,
	}
	req.CalculateHeaderLength()
	req.CalculateBodyLength()
	return message2.EncodeReq(req)
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
