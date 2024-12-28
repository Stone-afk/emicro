package opentelemetry

import (
	"context"
	"emicro/observability"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ClientInterceptorBuilder struct {
	port       int
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func (b *ClientInterceptorBuilder) BuildUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	address := observability.GetOutboundIP()
	if b.port != 0 {
		address = fmt.Sprintf("%s:%d", address, b.port)
	}
	//propagator := b.propagator
	//if b.propagator == nil {
	//	// 这个是全局
	//	propagator = otel.GetTextMapPropagator()
	//}
	//tracer := b.tracer
	//if tracer == nil {
	//	tracer = otel.Tracer("emicro/observability/opentelemetry")
	//}
	attrs := []attribute.KeyValue{
		semconv.RPCSystemKey.String("grpc"),
		attribute.Key("rpc.grpc.kind").String("unary"),
		attribute.Key("rpc.component").String("client"),
	}
	return func(ctx context.Context, method string, req,
		reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		ctx, span := b.tracer.Start(ctx, method,
			trace.WithAttributes(attrs...),
			trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		defer func() {
			if err != nil {
				span.SetAttributes(semconv.RPCGRPCStatusCodeKey.Int64(int64(codes.Error)))
				span.SetStatus(codes.Error, "client failed")
				span.RecordError(err)
			} else {
				span.SetStatus(codes.Ok, "OK")
			}
			span.End()
		}()
		// inject 过程
		// 要把跟 trace 有关的链路元数据，传递到服务端
		ctx = b.inject(ctx)
		err = invoker(ctx, method, req, reply, cc, opts...)
		return
	}
}

//func (b *ClientInterceptorBuilder) BuildUnary() grpc.UnaryClientInterceptor {
//	address := observability.GetOutboundIP()
//	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
//		ctx, span := b.Tracer.Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
//		span.SetAttributes(attribute.String("address", address))
//		ctx = b.inject(ctx)
//		defer func() {
//			if err != nil {
//				span.SetStatus(codes.Error, "client failed")
//				span.RecordError(err)
//			}
//			span.End()
//		}()
//		err = invoker(ctx, method, req, reply, cc, opts...)
//		return
//	}
//}

func (b *ClientInterceptorBuilder) inject(ctx context.Context) context.Context {
	// 先看 ctx 里面有没有元数据
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{})
	}
	// 把元数据放回去 ctx，具体怎么放，放什么内容，由 propagator 决定
	b.propagator.Inject(ctx, propagation.HeaderCarrier(md))
	return metadata.NewOutgoingContext(ctx, md)

}

func NewClientInterceptorBuilder(port int, tracer trace.Tracer, propagator propagation.TextMapPropagator) *ClientInterceptorBuilder {
	return &ClientInterceptorBuilder{port: port, tracer: tracer, propagator: propagator}
}
