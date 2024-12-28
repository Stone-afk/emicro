package roundrobin

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
	"testing"
)

func TestPicker_Pick(t *testing.T) {
	testCases := []struct {
		name            string
		p               *Picker
		wantErr         error
		wantSubConn     *conn
		wantPickerIndex uint64
	}{
		{
			name: "start",
			p: &Picker{
				cnt: 0,
				connections: []*conn{
					{address: resolver.Address{Addr: "127.0.0.1:8080"}},
					{address: resolver.Address{Addr: "127.0.0.1:8081"}},
				},
				filter: func(info balancer.PickInfo, address resolver.Address) bool {
					return true
				},
			},

			wantSubConn:     &conn{address: resolver.Address{Addr: "127.0.0.1:8080"}},
			wantPickerIndex: 0,
		},
		{
			name: "end",
			p: &Picker{
				cnt: 1,
				connections: []*conn{
					{address: resolver.Address{Addr: "127.0.0.1:8080"}},
					{address: resolver.Address{Addr: "127.0.0.1:8081"}},
				},
				filter: func(info balancer.PickInfo, address resolver.Address) bool {
					return true
				},
			},

			wantSubConn:     &conn{address: resolver.Address{Addr: "127.0.0.1:8081"}},
			wantPickerIndex: 1,
		},
		{
			name: "no connections",
			p: &Picker{
				cnt:         0,
				connections: []*conn{},
				filter: func(info balancer.PickInfo, address resolver.Address) bool {
					return true
				},
			},
			wantErr: balancer.ErrNoSubConnAvailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cnt := tc.p.cnt
			res, err := tc.p.Pick(balancer.PickInfo{})
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantSubConn.address.Addr, res.SubConn.(conn).address.Addr)
			assert.NotNil(t, res.Done)
			idx := cnt % uint64(len(tc.p.connections))
			assert.Equal(t, tc.wantPickerIndex, idx)
		})
	}
}
