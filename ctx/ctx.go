package ctx

import (
	"net/http"

	"github.com/lucagez/qron/sqlc"
)

func ExecutorSetterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner := r.Header.Get("x-owner")
		if owner == "" {
			owner = "default"
		}

		ctx := sqlc.NewCtx(r.Context(), owner)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
