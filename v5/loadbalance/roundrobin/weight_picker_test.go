package roundrobin

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
	"testing"
)

func TestWeightBalancer_Pick(t *testing.T) {
	b := &WeightPicker{
		connections: []*weightConn{
			{
				name:            "weight-5",
				weight:          5,
				efficientWeight: 5,
				currentWeight:   5,
			},
			{
				name:            "weight-4",
				weight:          4,
				efficientWeight: 4,
				currentWeight:   4,
			},
			{
				name:            "weight-3",
				weight:          3,
				efficientWeight: 3,
				currentWeight:   3,
			},
		},
		filter: func(info balancer.PickInfo, address resolver.Address) bool {
			return true
		},
	}
	pickRes, err := b.Pick(balancer.PickInfo{})
	require.NoError(t, err)
	assert.Equal(t, "weight-5", pickRes.SubConn.(*weightConn).name)

	pickRes, err = b.Pick(balancer.PickInfo{})
	require.NoError(t, err)
	assert.Equal(t, "weight-4", pickRes.SubConn.(*weightConn).name)

	pickRes, err = b.Pick(balancer.PickInfo{})
	require.NoError(t, err)
	assert.Equal(t, "weight-3", pickRes.SubConn.(*weightConn).name)

	pickRes, err = b.Pick(balancer.PickInfo{})
	require.NoError(t, err)
	assert.Equal(t, "weight-5", pickRes.SubConn.(*weightConn).name)

	pickRes, err = b.Pick(balancer.PickInfo{})
	require.NoError(t, err)
	assert.Equal(t, "weight-4", pickRes.SubConn.(*weightConn).name)

	pickRes.Done(balancer.DoneInfo{})
	// 断言这里面 efficient weight 是变化了的
}
