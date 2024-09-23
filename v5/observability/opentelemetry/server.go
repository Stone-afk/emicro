package opentelemetry

import (
	"context"
	"emicro/v5/observability"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const instrumentationName = "emicro/observability/opentelemetry"

type ServerInterceptorBuilder struct {
	port       int
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func (b *ServerInterceptorBuilder) BuildServerInterceptorBuilder() grpc.UnaryServerInterceptor {
	if b.tracer == nil {
		b.tracer = otel.GetTracerProvider().Tracer(instrumentationName)
	}
	address := observability.GetOutboundIP()
	if b.port != 0 {
		address = fmt.Sprintf("%s:%d", address, b.port)
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, span := b.tracer.Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		ctx = b.extract(ctx)
		// 这里可以记录非常多的数据，一般来说可以考虑机器本身的信息，例如 ip，端口
		// 也可以考虑进一步记录和请求有关的信息，例如业务 ID
		span.SetAttributes(attribute.String("address", address))
		defer func() {
			if err != nil {
				// 在使用 err.String()
				span.SetStatus(codes.Error, "server failed")
				span.RecordError(err)
			}
			span.End()
		}()
		resp, err = handler(ctx, req)
		return
	}
}

func (b *ServerInterceptorBuilder) extract(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{})
	}
	return b.propagator.Extract(ctx, propagation.HeaderCarrier(md))
}

func NewServerInterceptorBuilder(port int, tracer trace.Tracer, propagator propagation.TextMapPropagator) *ServerInterceptorBuilder {
	return &ServerInterceptorBuilder{port: port, tracer: tracer, propagator: propagator}
}
