package webserver

import (
	"net/http"

	"internal/shared/requestctx"
)

// CorrelationID returns middleware that extracts the X-Correlation-ID header
// from incoming requests and injects it into the request context via
// requestctx.WithCorrelationID. Handlers no longer need to extract this
// header manually — r.Context() already carries the correlation ID.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cid := r.Header.Get("X-Correlation-ID"); cid != "" {
			r = r.WithContext(requestctx.WithCorrelationID(r.Context(), cid))
		}
		next.ServeHTTP(w, r)
	})
}
