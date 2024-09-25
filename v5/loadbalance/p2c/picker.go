package p2c

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
	"math"
	"time"
)

// This package is a client load loadbalance. The algorithm used is p2c+ewma
// P2c is either
// Ewma index moving weighted average (reflecting the average value over a period of time)

const (
	// Name is the name of p2c loadbalance.
	Name            = "p2c_ewma"
	initSuccess     = 1000
	throttleSuccess = initSuccess / 2
	// if statistic not collected,we add a big lag penalty to endpoint
	penalty   = int64(math.MaxInt32)
	forcePick = int64(time.Second)
	pickTimes = 3
	// default value from finagle
	decayTime   = int64(time.Second * 10)
	logInterval = time.Minute
)

var emptyPickResult balancer.PickResult

type PickerBuilder struct{}

type Conn struct {
	balancer.SubConn
	address resolver.Address
}
