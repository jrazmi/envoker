// Package telemetry provides support for initializing the telemetry system.
package telemetry

import (
	"context"
	"time"

	"github.com/jrazmi/envoker/sdk/cryptids"
)

type telKey int

const (
	traceIDKey telKey = iota + 1
)

type TraceValues struct {
	TraceID    string
	Now        time.Time
	StatusCode int
}

type Telemetry struct{}

// Creates a new telemetry instance
func NewTelemetry() Telemetry {

	return Telemetry{}
}

func (t Telemetry) SetTraceID(ctx context.Context) context.Context {
	tid, err := cryptids.GenerateID()
	if err != nil {
		return context.WithValue(ctx, traceIDKey, "--------NOTRACE--------")
	}
	return context.WithValue(ctx, traceIDKey, tid)
}
func (t Telemetry) GetTraceID(ctx context.Context) string {
	v, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return "--------NOTRACE--------"
	}

	return v
}
