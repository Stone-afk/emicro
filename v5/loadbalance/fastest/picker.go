package fastest

import (
	"emicro/v5/loadbalance"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const Fastest = "FASTEST"

var (
	_ balancer.Picker    = (*Picker)(nil)
	_ base.PickerBuilder = (*PickerBuilder)(nil)
)

type PickerBuilder struct {
	// prometheus 的地址
	Endpoint string
	Query    string
	Filter   loadbalance.Filter
	// 刷新响应时间的间隔
	Interval time.Duration
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	connections := make([]*conn, 0, len(info.ReadySCs))
	for con, val := range info.ReadySCs {
		connections = append(connections, &conn{
			SubConn: con,
			address: val.Address,
		})
	}
	filter := b.Filter
	if filter == nil {
		filter = func(info balancer.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	return &Picker{
		connections: connections,
		filter:      filter,
	}
}

func (b *PickerBuilder) Name() string {
	return Fastest
}

type Picker struct {
	connections []*conn
	mutex       sync.RWMutex
	filter      loadbalance.Filter
	lastSync    time.Time
	endpoint    string
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mutex.Lock()
	if len(p.connections) == 0 {
		p.mutex.RUnlock()
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	var res *conn
	for _, c := range p.connections {
		if !p.filter(info, c.address) {
			continue
		}
		// 过滤最快响应时间
		if res == nil {
			res = c
		} else if res.respDuration > c.respDuration {
			res = c
		}
	}
	if res == nil {
		p.mutex.RUnlock()
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	p.mutex.RUnlock()
	return balancer.PickResult{
		SubConn: res,
		Done: func(info balancer.DoneInfo) {
		},
	}, nil
}

func (p *Picker) updateRespTime(endpoint, query string) {
	// 这里很难容错，即如果刷新响应时间失败该怎么办
	httpResp, err := http.Get(fmt.Sprintf("%s/api/v1/query?query=%s", endpoint, query))
	if err != nil {
		// 这里难处理，可以考虑记录错误，然后等下一次
		// 可以考虑中断
		// 也可以重试一定次数之后中断
		log.Fatalln("查询 prometheus 失败", err)
		return
	}
	// body, err := ioutil.ReadAll(httpResp.Body)
	//if err != nil {
	//	return
	//}
	//log.Println(string(body))
	decoder := json.NewDecoder(httpResp.Body)
	var resp response
	err = decoder.Decode(&resp)
	if err != nil {
		// 这里难处理，可以考虑记录错误，然后等下一次
		// 可以考虑中断
		// 也可以重试一定次数之后中断
		log.Fatalln("反序列化 http 响应失败", err)
		return
	}
	if resp.Status != "success" {
		// 查询返回错误结果
		log.Fatalln("失败的响应", err)
		return
	}
	for _, promRes := range resp.Data.Result {
		address, ok := promRes.Metric["address"]
		if !ok {
			return
		}
		for _, c := range p.connections {
			if c.address.Addr == address {
				ms, er := strconv.ParseInt(promRes.Value[1].(string), 10, 64)
				if er != nil {
					continue
				}
				c.respDuration = time.Duration(ms) * time.Millisecond
			}
		}
	}

}

type conn struct {
	balancer.SubConn
	address resolver.Address
	// response time
	respDuration time.Duration
}

type response struct {
	Status string `json:"status"`
	Data   data   `json:"data"`
}

type data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}
