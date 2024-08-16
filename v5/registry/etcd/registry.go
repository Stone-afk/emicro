package etcd

import (
	"context"
	"emicro/v5/registry"
	"encoding/json"
	"fmt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"sync"
)

var typesMap = map[mvccpb.Event_EventType]registry.EventType{
	mvccpb.PUT:    registry.EventTypeAdd,
	mvccpb.DELETE: registry.EventTypeDelete,
}

var _ registry.Registry = (*Registry)(nil)

type Registry struct {
	client       *clientv3.Client
	sess         *concurrency.Session
	mutex        sync.RWMutex
	SessTimeout  int64
	watchCancels []func()
}

func (r *Registry) Close() error {
	r.mutex.Lock()
	cancels := r.watchCancels
	r.watchCancels = nil
	r.mutex.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	// r.client.Close()
	// 因为 client 是外面传进来的，所以我们这里不能关掉它。它可能被其它的人使用着
	return r.sess.Close()
}

func (r *Registry) Register(ctx context.Context, ins registry.ServiceInstance) error {
	val, err := json.Marshal(ins)
	if err != nil {
		return err
	}
	// 这个 key 也可以做成可配置的
	// ctx = clientv3.WithRequireLeader(ctx)
	// 准备 key value 和租约
	// TODO 手工管理租约，要考虑续约间隔，续约时长，续约容错，续约容错的过程对服务发现的影响
	// lease := clientv3.NewLease(r.client)
	// lease.KeepAlive()
	// _, err = r.client.Put(ctx, instanceKey, string(val), clientv3.WithLease(lease.))

	_, err = r.client.Put(ctx, r.instanceKey(ins), string(val), clientv3.WithLease(r.sess.Lease()))
	return err
}

func (r *Registry) Unregister(ctx context.Context, ins registry.ServiceInstance) error {
	_, err := r.client.Delete(ctx, r.instanceKey(ins))
	return err
}

func (r *Registry) ListServices(ctx context.Context, serviceName string) ([]registry.ServiceInstance, error) {
	resp, err := r.client.Get(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	res := make([]registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var si registry.ServiceInstance
		err = json.Unmarshal(kv.Value, &si)
		if err != nil {
			return nil, err
		}
		res = append(res, si)
	}
	return res, nil
}

func (r *Registry) Subscribe(serviceName string) (<-chan registry.Event, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r.mutex.Lock()
	r.watchCancels = append(r.watchCancels, cancel)
	r.mutex.Unlock()
	ctx = clientv3.WithRequireLeader(ctx)
	watchResp := r.client.Watch(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	res := make(chan registry.Event)
	go func() {
		for {
			select {
			case resp := <-watchResp:
				if resp.Err() != nil {
					//return
					continue
				}
				if resp.Canceled {
					return
				}
				//for range resp.Events {
				//	res <- registry.Event{}
				//}
				for _, event := range resp.Events {
					res <- registry.Event{
						Type: typesMap[event.Type],
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return res, nil
}

func (r *Registry) instanceKey(ins registry.ServiceInstance) string {
	return fmt.Sprintf("/emicro/%s/%s", ins.Name, ins.Address)
}

func (r *Registry) serviceKey(serviceName string) string {
	return fmt.Sprintf("/emicro/%s", serviceName)
}

func NewRegistry(c *clientv3.Client) (*Registry, error) {
	// 没有设置 ttl，所以默认是 60 秒，这个可以做成可配置的
	//sess, err := concurrency.NewSession(c, concurrency.WithTTL(111))
	sess, err := concurrency.NewSession(c)
	if err != nil {
		return nil, err
	}
	return &Registry{
		client: c,
		sess:   sess,
	}, nil
}
