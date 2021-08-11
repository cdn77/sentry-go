package sentry

import (
	"fmt"

	"github.com/cdn77/sentry-go/internal/crypto/randutil"
)

// A TracesSampler makes sampling decisions for spans.
//
// In addition to the sampling context passed to the Sample method,
// implementations may keep and use internal state to make decisions.
//
// Sampling is one of the last steps when starting a new span, such that the
// sampler can inspect most of the state of the span to make a decision.
//
// Implementations must be safe for concurrent use by multiple goroutines.
type TracesSampler interface {
	Sample(ctx SamplingContext) Sampled
}

// Implementation note:
//
// TracesSampler.Sample return type is Sampled (instead of bool or float64), so
// that we can compose samplers by letting a sampler return SampledUndefined to
// defer the decision to the next sampler.
//
// For example, a hypothetical InheritFromParentSampler would return
// SampledUndefined if there is no parent span in the SamplingContext, deferring
// the sampling decision to another sampler, like a UniformSampler.
//
// var _ TracesSampler = sentry.TracesSamplers{
// 	sentry.InheritFromParentSampler,
// 	sentry.UniformTracesSampler(0.1),
// }
//
// Another example, we can provide a sampler that returns SampledFalse if the
// SamplingContext matches some condition, and SampledUndefined otherwise:
//
// var _ TracesSampler = sentry.TracesSamplers{
// 	sentry.IgnoreTransaction(regexp.MustCompile(`^\w+ /(favicon.ico|healthz)`),
// 	sentry.InheritFromParentSampler,
// 	sentry.UniformTracesSampler(0.1),
// }
//
// If after running all samplers the decision is still undefined, the
// span/transaction is not sampled.

// A SamplingContext is passed to a TracesSampler to determine a sampling
// decision.
//
// TODO(tracing): possibly expand SamplingContext to include custom /
// user-provided data.
type SamplingContext struct {
	Span   *Span // The current span, always non-nil.
	Parent *Span // The parent span, may be nil.
}

// The TracesSample type is an adapter to allow the use of ordinary
// functions as a TracesSampler.
type TracesSampler func(ctx SamplingContext) float64

func (f TracesSampler) Sample(ctx SamplingContext) float64 {
	return f(ctx)
}
