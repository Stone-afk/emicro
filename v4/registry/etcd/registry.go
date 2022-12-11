package etcd

import (
	"context"
	"emicro/v4/registry"
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

type Registry struct {
	client      *clientv3.Client
	sess        *concurrency.Session
	mutex       sync.RWMutex
	watchCancel []func()
}

func NewRegistry(c *clientv3.Client) (*Registry, error) {
	// 没有设置 ttl，所以默认是 60 秒，这个可以做成可配置的
	//sess, err := concurrency.NewSession(c, concurrency.WithTTL(111))
	sess, err := concurrency.NewSession(c)
	if err != nil {
		return nil, err
	}
	return &Registry{
		sess:   sess,
		client: c,
	}, nil
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
	_, err = r.client.Put(ctx, r.instanceKey(ins),
		string(val), clientv3.WithLease(r.sess.Lease()))
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
	ctx = clientv3.WithRequireLeader(ctx)
	r.mutex.Lock()
	r.watchCancel = append(r.watchCancel, cancel)
	r.mutex.Unlock()
	watchCh := r.client.Watch(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	res := make(chan registry.Event)
	go func() {
		for {
			select {
			case resp := <-watchCh:
				if resp.Canceled {
					close(res)
					return
				}
				if resp.Err() != nil {
					continue
				}
				for _, event := range resp.Events {
					var ins registry.ServiceInstance
					err := json.Unmarshal(event.Kv.Value, &ins)
					if err != nil {
						// 忽略这个事件吗？还是上报error，怎么上报 error 呢？
						// 忽略
						// continue
						select {
						case res <- registry.Event{}:
						// case <- r.close:
						case <-ctx.Done():
							close(res)
							return
						}
						continue
					}
					select {
					case res <- registry.Event{
						Type:     typesMap[event.Type],
						Insrance: ins,
					}:
					// case <- r.close:
					case <-ctx.Done():
						close(res)
						return
					}
				}
			case <-ctx.Done():
				close(res)
				return
			}
		}
	}()
	return res, nil
}

func (r *Registry) Close() error {
	r.mutex.Lock()
	for _, cancel := range r.watchCancel {
		cancel()
	}
	r.mutex.Unlock()
	// r.client.Close()
	// 因为 client 是外面传进来的，所以我们这里不能关掉它。它可能被其它的人使用着
	return r.sess.Close()
}

func (r *Registry) instanceKey(ins registry.ServiceInstance) string {
	return fmt.Sprintf("/emicro/v4/%s/%s", ins.Name, ins.Address)
}

func (r *Registry) serviceKey(serviceName string) string {
	return fmt.Sprintf("/emicro/v4/%s", serviceName)
}
