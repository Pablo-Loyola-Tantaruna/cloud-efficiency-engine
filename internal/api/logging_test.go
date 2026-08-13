package api

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware_ShouldLogCompletedRequest(
	t *testing.T,
) {

	// Arrange

	var buffer bytes.Buffer

	logger :=
		slog.New(
			slog.NewJSONHandler(
				&buffer,
				nil,
			),
		)

	handler :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				w.WriteHeader(
					http.StatusOK,
				)
			},
		)

	middleware :=
		loggingMiddleware(
			logger,
			handler,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		)

	request =
		request.WithContext(
			contextWithRequestID(
				request.Context(),
				"test-request-123",
			),
		)

	responseRecorder :=
		httptest.NewRecorder()

	// Act

	middleware.ServeHTTP(
		responseRecorder,
		request,
	)

	// Assert

	if responseRecorder.Code !=
		http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			responseRecorder.Code,
		)
	}

	logOutput :=
		buffer.String()

	if !strings.Contains(
		logOutput,
		"http_request_completed",
	) {

		t.Fatalf(
			"expected completion event in log: %s",
			logOutput,
		)
	}

	if !strings.Contains(
		logOutput,
		"test-request-123",
	) {

		t.Fatalf(
			"expected request ID in log: %s",
			logOutput,
		)
	}

	if !strings.Contains(
		logOutput,
		"/test",
	) {

		t.Fatalf(
			"expected request path in log: %s",
			logOutput,
		)
	}
}

func TestLoggingMiddleware_ShouldLogErrorStatus(
	t *testing.T,
) {

	// Arrange

	var buffer bytes.Buffer

	logger :=
		slog.New(
			slog.NewJSONHandler(
				&buffer,
				nil,
			),
		)

	handler :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				w.WriteHeader(
					http.StatusInternalServerError,
				)
			},
		)

	middleware :=
		loggingMiddleware(
			logger,
			handler,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		)

	request =
		request.WithContext(
			contextWithRequestID(
				request.Context(),
				"error-request-123",
			),
		)

	responseRecorder :=
		httptest.NewRecorder()

	// Act

	middleware.ServeHTTP(
		responseRecorder,
		request,
	)

	// Assert

	if responseRecorder.Code !=
		http.StatusInternalServerError {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			responseRecorder.Code,
		)
	}

	logOutput :=
		buffer.String()

	if !strings.Contains(
		logOutput,
		"error-request-123",
	) {

		t.Fatalf(
			"expected request ID in log: %s",
			logOutput,
		)
	}

	if !strings.Contains(
		logOutput,
		"500",
	) {

		t.Fatalf(
			"expected status 500 in log: %s",
			logOutput,
		)
	}
}

func contextWithRequestID(
	ctx context.Context,
	requestID string,
) context.Context {

	return context.WithValue(
		ctx,
		requestIDKey{},
		requestID,
	)
}
