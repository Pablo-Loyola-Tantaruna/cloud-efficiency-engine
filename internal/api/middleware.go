package api

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type requestIDKey struct{}

func requestIDFromContext(
	ctx context.Context,
) string {

	value :=
		ctx.Value(
			requestIDKey{},
		)

	requestID, ok :=
		value.(string)

	if !ok {
		return ""
	}

	return requestID
}

func requestIDMiddleware(
	next http.Handler,
) http.Handler {

	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			requestID :=
				r.Header.Get(
					"X-Request-ID",
				)

			if requestID == "" {

				requestID =
					fmt.Sprintf(
						"%d",
						time.Now().UnixNano(),
					)
			}

			ctx :=
				context.WithValue(
					r.Context(),
					requestIDKey{},
					requestID,
				)

			w.Header().Set(
				"X-Request-ID",
				requestID,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		},
	)
}
