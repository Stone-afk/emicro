package etcd

import (
	"emicro/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"testing"
)

func TestRegistry_Subscribe(t *testing.T) {
	testCases := []struct {
		name      string
		mock      func() clientv3.Watcher
		wantEvent registry.Event
		wantErr   error
	}{
		{
			// mock: func() clientv3.Watcher {
			// 	watcher := mocks.NewMockWatcher()
			// },
		},
	}
	for _, tc := range testCases {
		r := &Registry{
			client: &clientv3.Client{
				Watcher: tc.mock(),
			},
		}
		ch, err := r.Subscribe("service-name")
		assert.Equal(t, tc.wantErr, err)
		event := <-ch
		// log.Println(event)
		assert.Equal(t, tc.wantEvent, event)
		err = r.Close()
		require.NoError(t, err)
		_, ok := <-ch
		assert.False(t, ok)

	}
}
