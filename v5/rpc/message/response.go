package message

import "encoding/binary"

// Response ->
type Response struct {
	// 响应头长度
	HeadLength uint32
	// 响应体长度
	BodyLength uint32
	//消息ID
	MessageId uint32
	// 协议版本
	Version uint8
	// 压缩算法
	Compresser uint8
	// 序列化协议
	Serializer uint8
	// 错误
	Error []byte
	// 你要区分业务 error 还是非业务 error
	// BizError []byte // 代表的是业务返回的 error
	// 协议 响应体 / 响应数据
	Data []byte
}

func EncodeResp(resp *Response) []byte {
	bs := make([]byte, resp.HeadLength+resp.BodyLength)
	// 1. 写入 HeadLength，四个字节
	binary.BigEndian.PutUint32(bs[:4], resp.HeadLength)
	// 2. 写入 BodyLength 四个字节
	binary.BigEndian.PutUint32(bs[4:8], resp.BodyLength)
	// 3. 写入 message id, 四个字节
	binary.BigEndian.PutUint32(bs[8:12], resp.MessageId)

	// 4. 写入 version，因为本身就是一个字节，所以不用进行编码了
	bs[12] = resp.Version
	// 5. 写入压缩算法
	bs[13] = resp.Compresser
	// 6. 写入序列化协议
	bs[14] = resp.Serializer

	// 7. 写入 error, 写入 Eorror 后不 +1 -> (cur[len(resp.Error)+1:]) 是因为
	// 直接取头部长度就区分了 head 和 body ， 而 Error 的前一个参数 Serializer 也刚好为 1 字节
	cur := bs[15:]
	copy(cur, resp.Error)
	cur = cur[len(resp.Error):]

	// 8. 剩下的数据
	copy(cur, resp.Data)
	return bs
}

func DecodeResp(bs []byte) *Response {
	resp := &Response{}
	// 1. 读取 HeadLength
	resp.HeadLength = binary.BigEndian.Uint32(bs[:4])
	// 2. 读取 BodyLength
	resp.BodyLength = binary.BigEndian.Uint32(bs[4:8])
	// 3. 读取 message id
	resp.MessageId = binary.BigEndian.Uint32(bs[8:12])

	// 4. 读取 Version
	resp.Version = bs[12]
	// 5. 读取压缩算法
	resp.Compresser = bs[13]
	// 6. 读取序列化协议
	resp.Serializer = bs[14]

	// 7. error 信息
	resp.Error = bs[15:resp.HeadLength]

	// 剩下的就是数据了
	resp.Data = bs[resp.HeadLength:]
	return resp
}

func (resp *Response) CalculateHeaderLength() {
	resp.HeadLength = 15 + uint32(len(resp.Error))
}

func (resp *Response) CalculateBodyLength() {
	resp.BodyLength = uint32(len(resp.Data))
}
