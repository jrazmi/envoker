package mid

import (
	"context"
	"net/http"

	"github.com/jrazmi/envoker/bridge/scaffolding/metrics"
	"github.com/jrazmi/envoker/infrastructure/web"
)

// Metrics updates program counters.
func Metrics() web.Middleware {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) web.Encoder {
			ctx = metrics.Set(ctx)

			resp := next(ctx, r)

			n := metrics.AddRequests(ctx)

			if n%1000 == 0 {
				metrics.AddGoroutines(ctx)
			}

			if isError(resp) != nil {
				metrics.AddErrors(ctx)
			}

			return resp
		}
	}
}
