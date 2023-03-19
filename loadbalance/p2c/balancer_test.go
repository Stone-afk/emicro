package p2c

import (
	"context"
	xmath "emicro/internal/utils/xmath"
	xstring "emicro/internal/utils/xstring"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
	"runtime"
	"strconv"
	"sync"
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
			picker := builder.Build(base.PickerBuildInfo{
				ReadySCs: ready,
			})
			var wg sync.WaitGroup
			wg.Add(total)
			for i := 0; i < total; i++ {
				result, err := picker.Pick(balancer.PickInfo{
					FullMethodName: "/",
					Ctx:            context.Background(),
				})
				assert.Equal(t, tc.err, err)

				if tc.err != nil {
					return
				}
				if i%100 == 0 {
					err = status.Error(codes.DeadlineExceeded, "deadline")
				}
				go func() {
					runtime.Gosched()
					result.Done(balancer.DoneInfo{
						Err: err,
					})
					wg.Done()
				}()
			}
			wg.Wait()
			dist := make(map[any]int)
			conns := picker.(*Picker).conns
			for _, conn := range conns {
				dist[conn.address.Addr] = int(conn.requests)
			}
			entropy := xmath.CalcEntropy(dist)
			assert.True(t, entropy > tc.threshold, fmt.Sprintf("entropy is %f, less than %f",
				entropy, tc.threshold))
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
