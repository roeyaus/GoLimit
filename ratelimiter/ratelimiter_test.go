package ratelimiter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	_, err := NewRateLimiter(1, 0)
	assert.Error(t, err)
	_, err = NewRateLimiter(0, 1)
	assert.Error(t, err)
	_, err = NewRateLimiter(1, 1)
	assert.NoError(t, err)
}

func TestRequestHandlerSuccessAndLimit(t *testing.T) {
	rl, err := NewRateLimiter(1, 1)
	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	rl.HandleRequest(handler).ServeHTTP(rr, req)
	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// This one should fail with 429
	rl.HandleRequest(handler).ServeHTTP(rr, req)
	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusTooManyRequests)
	}
}
