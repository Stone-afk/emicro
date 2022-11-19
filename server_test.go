package emicro

import (
	"context"
	"emicro/message"
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
		server.RegisterService(tc.service)
		err := server.handleConn(tc.conn)
		require.NoError(t, err)
		// 比较写入的数据，去掉长度字段
		data := tc.conn.writeData[8:]
		resp := &message.Response{}
		err = json.Unmarshal(data, resp)
		require.NoError(t, err)
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
	}
	reqData, err := json.Marshal(req)
	require.NoError(t, err)
	return EncodeMsg(reqData)
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

// 客户端
// 1. 首先反射拿到 Request，核心是服务名字，方法名字和参数
// 2. 将 Request 进行编码，要注意序列化并且加上长度字段
// 3. 使用连接池，或者一个连接，把请求发过去
// 4. 从连接里面读取响应，解析成结构体

// 服务端
// 1. 启动一个服务器，监听一个端口
// 2. 读取长度字段，再根据长度，读完整个消息
// 3. 解析成 Request
// 4. 查找服务，会对应的方法
// 5. 构造方法对应的输入
// 6. 反射执行调用
// 7. 编码响应
// 8. 写回响应
