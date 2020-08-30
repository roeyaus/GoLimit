package ratelimiter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/elliotchance/redismock/v7"
	"github.com/go-redis/redis/v7"
	cache "github.com/roeyaus/airtasker/cache/redis"
	"github.com/stretchr/testify/assert"
)

func getMockRedisClient(windowInSeconds int, maxRequestsPerWindow int) *cache.RedisClient {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	mc := redismock.NewNiceMock(client)

	return &cache.RedisClient{RedisClient: mc, WindowInSeconds: windowInSeconds, MaxRequestsPerWindow: maxRequestsPerWindow}
}

func TestNewRateLimiter(t *testing.T) {
	_, err := NewRateLimiterWithCacheClient(1, 0, getMockRedisClient(1, 0))
	assert.Error(t, err)
	_, err = NewRateLimiterWithCacheClient(0, 1, getMockRedisClient(1, 0))
	assert.Error(t, err)
	_, err = NewRateLimiterWithCacheClient(1, 1, getMockRedisClient(1, 0))
	assert.NoError(t, err)
}

func TestRequestHandlerSuccessAndLimit(t *testing.T) {
	rl, err := NewRateLimiterWithCacheClient(1, 1, getMockRedisClient(1, 1))
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
	rl.HandleRequestsByIP(handler).ServeHTTP(rr, req)
	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// This one should fail with 429
	rl.HandleRequestsByIP(handler).ServeHTTP(rr, req)
	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusTooManyRequests)
	}
}
