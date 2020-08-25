package ratelimiter

import (
	"net/http"

	"github.com/roeyaus/airtasker/redis"
)

type RateLimiterResponse struct {
	StatusCode    int
	StatusMessage string
}

//RateLimiterInt is an interface for implementing a rate-limiter with a certain strategy
type RateLimiterInt interface {
	//This function will return
	HandleRequestIfAllowed(handler http.Handler) http.Handler
	Allowed()
}

type RateLimiter struct {
	redisClient *redis.RedisClient
}

func NewRateLimiter() (*RateLimiter, error) {
	if c, err := redis.GetClient(); err != nil {
		return nil, err
	} else {
		r := &RateLimiter{redisClient: c}
		return r, nil
	}
}

func HandleRequestIfAllowed(handler http.Handler) http.Handler {
	return http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

	})
	return RateLimiterResponse{StatusCode: 200, StatusMessage: "cool"}, nil
}
