package executor

import (
	"context"
	"log"
	"net/http"
)

type ctxKey struct{}

var key = ctxKey{}

func ExecutorSetterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: naming? should be `subscriber`?
		executor := r.Header.Get("x-executor")
		if executor == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(http.StatusText(http.StatusUnprocessableEntity)))
			return
		}

		ctx := NewCtx(r.Context(), executor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func FromCtx(ctx context.Context) string {
	executor, ok := ctx.Value(key).(string)
	if !ok {
		log.Fatal("executor not found in context")
	}
	return executor
}

func NewCtx(ctx context.Context, executor string) context.Context {
	return context.WithValue(ctx, key, executor)
}
