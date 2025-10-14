package mid

import (
	"context"
	"errors"
	"net/http"
	"path"

	"github.com/jrazmi/envoker/bridge/scaffolding/errs"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/logger"
)

// Errors handles errors coming out of the call chain.
func Errors(log *logger.Logger) web.Middleware {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) web.Encoder {
			resp := next(ctx, r)
			err := isError(resp)
			if err == nil {
				return resp
			}

			var appErr *errs.Error
			if !errors.As(err, &appErr) {
				appErr = errs.Newf(errs.Internal, "Internal Server Error")
			}

			log.ErrorContext(ctx, "handled error during request",
				"err", err,
				"source_err_file", path.Base(appErr.FileName),
				"source_err_func", path.Base(appErr.FuncName))

			if appErr.Code == errs.InternalOnlyLog {
				appErr = errs.Newf(errs.Internal, "Internal Server Error")
			}

			// Return the error as the response - your existing errs.Error
			// already implements Encoder, so this works perfectly
			return appErr
		}
	}
}
