package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDMiddleware_ShouldPreserveExistingRequestID3(
	t *testing.T,
) {

	// Arrange

	expectedRequestID :=
		"test-request-123"

	handler :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				requestID :=
					requestIDFromContext(
						r.Context(),
					)

				if requestID !=
					expectedRequestID {

					t.Fatalf(
						"expected request ID %s in context, got %s",
						expectedRequestID,
						requestID,
					)
				}

				w.WriteHeader(
					http.StatusOK,
				)
			},
		)

	middleware :=
		requestIDMiddleware(
			handler,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		)

	request.Header.Set(
		"X-Request-ID",
		expectedRequestID,
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

	actualRequestID :=
		responseRecorder.Header().Get(
			"X-Request-ID",
		)

	if actualRequestID !=
		expectedRequestID {

		t.Fatalf(
			"expected response request ID %s, got %s",
			expectedRequestID,
			actualRequestID,
		)
	}
}

func TestRequestIDMiddleware_ShouldGenerateRequestID2(
	t *testing.T,
) {

	// Arrange

	handler :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				requestID :=
					requestIDFromContext(
						r.Context(),
					)

				if requestID == "" {

					t.Fatal(
						"expected request ID in context",
					)
				}

				w.WriteHeader(
					http.StatusOK,
				)
			},
		)

	middleware :=
		requestIDMiddleware(
			handler,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
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

	requestID :=
		responseRecorder.Header().Get(
			"X-Request-ID",
		)

	if requestID == "" {

		t.Fatal(
			"expected generated request ID",
		)
	}

	if !isUUIDv4(requestID) {

		t.Fatalf(
			"expected UUID v4 request ID, got %s",
			requestID,
		)
	}
}

func TestRequestIDMiddleware_ShouldGenerateDifferentIDs(
	t *testing.T,
) {

	// Arrange

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
		requestIDMiddleware(
			handler,
		)

	// Act

	firstRequest :=
		httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		)

	firstResponse :=
		httptest.NewRecorder()

	middleware.ServeHTTP(
		firstResponse,
		firstRequest,
	)

	secondRequest :=
		httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		)

	secondResponse :=
		httptest.NewRecorder()

	middleware.ServeHTTP(
		secondResponse,
		secondRequest,
	)

	firstID :=
		firstResponse.Header().Get(
			"X-Request-ID",
		)

	secondID :=
		secondResponse.Header().Get(
			"X-Request-ID",
		)

	// Assert

	if firstID == "" ||
		secondID == "" {

		t.Fatal(
			"expected both requests to have request IDs",
		)
	}

	if firstID == secondID {

		t.Fatalf(
			"expected different request IDs, got %s",
			firstID,
		)
	}
}

func TestRequestIDFromContext_ShouldReturnEmptyWhenMissing(
	t *testing.T,
) {

	// Arrange

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		)

	// Act

	requestID :=
		requestIDFromContext(
			request.Context(),
		)

	// Assert

	if requestID != "" {

		t.Fatalf(
			"expected empty request ID, got %s",
			requestID,
		)
	}
}

func TestGenerateRequestID_ShouldReturnUUIDv4(
	t *testing.T,
) {

	// Act

	requestID :=
		generateRequestID()

	// Assert

	if !isUUIDv4(requestID) {

		t.Fatalf(
			"expected UUID v4, got %s",
			requestID,
		)
	}
}

func isUUIDv4(
	value string,
) bool {

	if len(value) != 36 {
		return false
	}

	if value[8] != '-' ||
		value[13] != '-' ||
		value[18] != '-' ||
		value[23] != '-' {

		return false
	}

	for index, character := range value {

		if index == 8 ||
			index == 13 ||
			index == 18 ||
			index == 23 {

			continue
		}

		if !strings.ContainsRune(
			"0123456789abcdefABCDEF",
			character,
		) {

			return false
		}
	}

	return value[14] == '4' &&
		(value[19] == '8' ||
			value[19] == '9' ||
			value[19] == 'a' ||
			value[19] == 'b' ||
			value[19] == 'A' ||
			value[19] == 'B')
}
