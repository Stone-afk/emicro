package p2c

import (
	"context"
	xstring "emicro/internal/utils/xstring"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"strconv"
	"testing"
)

func TestP2cPicker_PickNil(t *testing.T) {
	builder := new(PickerBuilder)
	picker := builder.Build(base.PickerBuildInfo{})
	_, err := picker.Pick(balancer.PickInfo{
		FullMethodName: "/",
		Ctx:            context.Background(),
	})
	assert.NotNil(t, err)
}

func TestP2cPicker_Pick(t *testing.T) {
	testCases := []struct {
		name       string
		candidates int
		err        error
		threshold  float64
	}{
		{
			name:       "empty",
			candidates: 0,
			err:        balancer.ErrNoSubConnAvailable,
		},
		{
			name:       "single",
			candidates: 1,
			threshold:  0.9,
		},
		{
			name:       "two",
			candidates: 2,
			threshold:  0.5,
		},
		{
			name:       "multiple",
			candidates: 100,
			threshold:  0.95,
		},
	}
	for _, tt := range testCases {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			const total = 10000
			builder := new(PickerBuilder)
			ready := make(map[balancer.SubConn]base.SubConnInfo)
			for i := 0; i < tc.candidates; i++ {
				ready[mockClientConn{
					id: xstring.Rand(),
				}] = base.SubConnInfo{
					Address: resolver.Address{
						Addr: strconv.Itoa(i),
					},
				}
			}

			_ = builder.Build(base.PickerBuildInfo{
				ReadySCs: ready,
			})
		})
	}

}

type mockClientConn struct {
	// add random string member to avoid map key equality.
	id string
}

func (m mockClientConn) GetOrBuildProducer(builder balancer.ProducerBuilder) (
	p balancer.Producer, close func()) {
	return builder.Build(m)
}

func (m mockClientConn) UpdateAddresses(addresses []resolver.Address) {}

func (m mockClientConn) Connect() {}
