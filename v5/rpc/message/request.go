package message

import (
	"bytes"
	"encoding/binary"
)

const (
	splitter     = '\n'
	pairSplitter = '\r'
)

// Request ->
type Request struct {
	// 头部
	// 消息头长度
	HeadLength uint32
	// 消息体长度
	BodyLength uint32
	// 消息 ID
	MessageId uint32
	// 版本，一个字节
	Version uint8
	// 压缩算法
	Compresser uint8
	// 序列化协议
	Serializer uint8

	// 服务名和方法名
	ServiceName string
	MethodName  string

	// 扩展字段，用于传递自定义元数据
	Meta map[string]string

	// 协议 请求体 / 请求数据
	Data []byte
}

func EncodeReq(req *Request) []byte {
	bs := make([]byte, req.HeadLength+req.BodyLength)
	// 1. 写入 HeadLength，四个字节
	binary.BigEndian.PutUint32(bs[:4], req.HeadLength)
	// 2. 写入 BodyLength 四个字节
	binary.BigEndian.PutUint32(bs[4:8], req.BodyLength)
	// 3. 写入 message id, 四个字节
	binary.BigEndian.PutUint32(bs[8:12], req.MessageId)

	// 4. 写入 version，因为本身就是一个字节，所以不用进行编码了
	bs[12] = req.Version
	// 5. 写入压缩算法
	bs[13] = req.Compresser
	// 6. 写入序列化协议
	bs[14] = req.Serializer

	cur := bs[15:]
	copy(cur, req.ServiceName)
	cur = cur[len(req.ServiceName):]
	cur[0] = splitter
	cur = cur[1:]
	copy(cur, req.MethodName)
	cur = cur[len(req.MethodName):]
	cur[0] = splitter
	cur = cur[1:]

	for key, value := range req.Meta {
		copy(cur, key)
		cur = cur[len(key):]
		cur[0] = pairSplitter
		cur = cur[1:]
		copy(cur, value)
		cur = cur[len(value):]
		cur[0] = splitter
		cur = cur[1:]
	}
	copy(cur, req.Data)
	return bs
}

func DecodeReq(bs []byte) *Request {
	req := &Request{}
	// 按照 EncodeReq 写下来
	// 1. 读取 HeadLength
	req.HeadLength = binary.BigEndian.Uint32(bs[:4])
	// 2. 读取 BodyLength
	req.BodyLength = binary.BigEndian.Uint32(bs[4:8])
	// 3. 读取 message id
	req.MessageId = binary.BigEndian.Uint32(bs[8:12])
	// 4. 读取 Version
	req.Version = bs[12]
	// 5. 读取压缩算法
	req.Compresser = bs[13]
	// 6. 读取序列化协议
	req.Serializer = bs[14]
	// 是头部剩余数据
	header := bs[15:req.HeadLength]
	// 7. 拆解服务名和方法名
	index := bytes.IndexByte(header, splitter)
	req.ServiceName = string(header[:index])
	// 加1 是为了跳掉分隔符
	header = header[index+1:]

	index = bytes.IndexByte(header, splitter)
	// 拆解方法名
	req.MethodName = string(header[:index])
	// 加1 是为了跳掉分隔符
	header = header[index+1:]
	index = bytes.IndexByte(header, splitter)
	if len(header) > 0 || index != -1 {
		meta := make(map[string]string, 4)
		// 切出来了
		for index != -1 {
			// 一个键值对
			pair := header[:index]
			// 切分 key-value
			// 我们使用 \r 来切分键值对
			pairIndex := bytes.IndexByte(pair, pairSplitter)
			key := string(pair[:pairIndex])
			// +1 也是为了跳掉分隔符
			value := string(pair[pairIndex+1:])
			meta[key] = value
			// 往前移动 +1 跳掉分隔符
			header = header[index+1:]
			index = bytes.IndexByte(header, splitter)
		}
	}
	// 9. 读取协议请求体数据
	if req.BodyLength != 0 {
		req.Data = bs[req.HeadLength:]
	}
	return req
}

func (req *Request) CalculateHeaderLength() {
	// 不要忘了分隔符
	headLength := 15 + len(req.ServiceName) + 1 + len(req.MethodName) + 1
	for key, value := range req.Meta {
		headLength += len(key)
		// key 和 value 之间的分隔符
		headLength++
		headLength += len(value)
		headLength++
		// 和下一个 key value 的分隔符
	}
	req.HeadLength = uint32(headLength)
}

func (req *Request) CalculateBodyLength() {
	req.BodyLength = uint32(len(req.Data))
}
