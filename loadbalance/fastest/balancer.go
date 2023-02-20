package fastest

import (
	"emicro/loadbalance"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const Fastest = "FASTEST"

type Picker struct {
	conns    []*conn
	mutex    sync.RWMutex
	filter   loadbalance.Filter
	lastSync time.Time
	endpoint string
}

func (p *Picker) Pick(info loadbalance.PickInfo) (loadbalance.PickResult, error) {
	p.mutex.Lock()
	if len(p.conns) == 0 {
		p.mutex.RUnlock()
		return loadbalance.PickResult{}, loadbalance.ErrNoSubConnAvailable
	}
	var res *conn
	for _, c := range p.conns {
		if !p.filter(info, c.address) {
			continue
		}
		// 过滤最快响应时间
		if res == nil {
			res = c
		} else if res.response > c.response {
			res = c
		}
	}
	p.mutex.RUnlock()

	return loadbalance.PickResult{
		SubConn: res.SubConn,
		Done: func(info loadbalance.DoneInfo) {
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
	//body, err := ioutil.ReadAll(httpResp.Body)
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

		for _, c := range p.conns {
			if c.address.Addr == address {
				ms, err := strconv.ParseInt(promRes.Value[1].(string), 10, 64)
				if err != nil {
					continue
				}
				c.response = time.Duration(ms) * time.Millisecond
			}
		}
	}
}

type PickerBuilder struct {
	Filter loadbalance.Filter
	// prometheus 的地址
	Endpoint string
	Query    string
	// 刷新响应时间的间隔
	Interval time.Duration
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) loadbalance.Picker {
	conns := make([]*conn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		conns = append(conns, &conn{
			SubConn: con,
			address: conInfo.Address,
			// 随便设置一个默认值。当然这个默认值会对初始的负载均衡有影响
			// 不过一段时间之后就没什么影响了
			response: time.Millisecond * 100,
		})
	}
	filter := b.Filter
	if filter == nil {
		filter = func(info loadbalance.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	res := &Picker{
		conns:  conns,
		filter: filter,
	}
	// 这里有一个很大的问题，就是我们这里不好怎么退出，因为 gRPC 不会调用 Close 方法
	// 可以考虑使用 runtime.SetFinalizer 来在 res 被回收的时候得到通知
	ch := make(chan struct{}, 1)
	runtime.SetFinalizer(res, func() {
		ch <- struct{}{}
	})
	go func() {
		ticker := time.NewTicker(b.Interval)
		for {
			select {
			case <-ticker.C:
				// 这里很难容错，即如果刷新响应时间失败该怎么办
				res.updateRespTime(b.Endpoint, b.Query)
			case <-ch:
				return
			}
		}
	}()
	return res
}

func (b *PickerBuilder) Name() string {
	return Fastest
}

type conn struct {
	loadbalance.SubConn
	address resolver.Address
	// response time
	response time.Duration
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
