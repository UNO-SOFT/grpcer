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

	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
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
