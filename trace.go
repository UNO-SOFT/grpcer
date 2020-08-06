// Copyright 2020 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package grpcer

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/instrumentation/grpctrace"
	setrace "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var OTelHTTPPropagators = propagation.New(
	propagation.WithExtractors(trace.DefaultHTTPPropagator(), trace.B3{}),
	propagation.WithInjectors(trace.DefaultHTTPPropagator(), trace.B3{}),
)

func OTelExtractHTTP(ctx context.Context, headers http.Header) context.Context {
	return propagation.ExtractHTTP(ctx, OTelHTTPPropagators, headers)
}
func OTelInjectHTTP(ctx context.Context, headers http.Header) {
	propagation.InjectHTTP(ctx, OTelHTTPPropagators, headers)
}

func OTelGRPCExtract(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) ([]kv.KeyValue, trace.SpanContext) {
	return grpctrace.Extract(ctx, metadata, opts...)
}

func OTelGRPCInject(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) {
	grpctrace.Inject(ctx, metadata, opts...)
}
func OTelStreamClientInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.StreamClientInterceptor {
	return grpctrace.StreamClientInterceptor(tracer, opts...)
}
func OTelStreamServerInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.StreamServerInterceptor {
	return grpctrace.StreamServerInterceptor(tracer, opts...)
}

func OTelUnaryClientInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.UnaryClientInterceptor {
	return grpctrace.UnaryClientInterceptor(tracer, opts...)
}
func OTelUnaryServerInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.UnaryServerInterceptor {
	return grpctrace.UnaryServerInterceptor(tracer, opts...)
}

type Tracer = trace.Tracer

func LogTracer(Log func(...interface{}) error, name string) Tracer {
	if Log == nil {
		return global.Tracer(name)
	}
	exporter := LogExporter{Log: Log}
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter),
	)
	if err != nil {
		panic(err)
	}
	return tp.Tracer(name)
}

type LogExporter struct {
	Log func(...interface{}) error
}

// ExportSpans writes SpanData in json format to stdout.
func (e LogExporter) ExportSpans(ctx context.Context, data []*setrace.SpanData) {
	for _, d := range data {
		e.ExportSpan(ctx, d)
	}
}

// ExportSpan writes a SpanData in json format to stdout.
func (e LogExporter) ExportSpan(ctx context.Context, data *setrace.SpanData) {
	/*
	   type SpanData struct {
	   	SpanContext  apitrace.SpanContext
	   	ParentSpanID apitrace.SpanID
	   	SpanKind     apitrace.SpanKind
	   	Name         string
	   	StartTime    time.Time
	   	// The wall clock time of EndTime will be adjusted to always be offset
	   	// from StartTime by the duration of the span.
	   	EndTime                  time.Time
	   	Attributes               []kv.KeyValue
	   	MessageEvents            []Event
	   	Links                    []apitrace.Link
	   	StatusCode               codes.Code
	   	StatusMessage            string
	   	HasRemoteParent          bool
	   	DroppedAttributeCount    int
	   	DroppedMessageEventCount int
	   	DroppedLinkCount         int

	   	// ChildSpanCount holds the number of child span created for this span.
	   	ChildSpanCount int

	   	// Resource contains attributes representing an entity that produced this span.
	   	Resource *resource.Resource

	   	// InstrumentationLibrary defines the instrumentation library used to
	   	// providing instrumentation.
	   	InstrumentationLibrary instrumentation.Library
	   }
	*/
	e.Log("msg", "trace", "parent", data.ParentSpanID, "kind", data.SpanKind, "name", data.Name,
		"spanID", data.SpanContext.SpanID, "traceID", data.SpanContext.TraceID, "start", data.StartTime, "end", data.EndTime,
		"attrs", data.Attributes, "events", data.MessageEvents, "links", data.Links,
		"statusCode", data.StatusCode, "statusMsg", data.StatusMessage,
	)
}
