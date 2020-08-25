package ratelimiter

import (
	"fmt"
	"net/http"

	cache "github.com/roeyaus/airtasker/cache/redis"
	"github.com/roeyaus/airtasker/utils"
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
	redisClient         *cache.RedisClient
	intervalInSeconds   int
	requestsPerInterval int
}

func NewRateLimiter(intervalInSeconds int, requestsPerInterval int) (*RateLimiter, error) {
	if c, err := cache.GetRedisClient(intervalInSeconds, requestsPerInterval); err != nil {
		return nil, err
	} else {
		r := &RateLimiter{redisClient: c}
		return r, nil
	}
}

func (l RateLimiter) IsRequestAllowedForIP(ip string) (bool, error) {
	numRequests, err := l.redisClient.GetRequestsWithinInterval(ip, l.intervalInSeconds)
	if err != nil {
		return false, fmt.Errorf("could'nt get number of requests made this interval because %v", err)
	}
	if numRequests >= l.requestsPerInterval {
		return false, nil
	}
	return true, nil
}

func (l RateLimiter) HandleRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//get user's IP
		ip := utils.GetIP(r)
		if allowed, err := l.IsRequestAllowedForIP(ip); err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return

		} else if !allowed {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
