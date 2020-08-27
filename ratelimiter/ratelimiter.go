package ratelimiter

import (
	"fmt"
	"net/http"

	"github.com/roeyaus/airtasker/cache"
	rediscache "github.com/roeyaus/airtasker/cache/redis"
	"github.com/roeyaus/airtasker/utils"
)

type RateLimiter struct {
	client cache.CacheClient
}

func NewRateLimiter(windowInSeconds int, maxRequestsPerWindow int) (*RateLimiter, error) {
	if maxRequestsPerWindow < 1 {
		return nil, fmt.Errorf("maxRequestsPerWindow must be > 0")
	}
	if windowInSeconds < 1 {
		return nil, fmt.Errorf("windowInSeconds must be > 0")
	}
	if c, err := rediscache.GetRedisClient("localhost:6379", "", 0, windowInSeconds, maxRequestsPerWindow); err != nil {
		return nil, err
	} else {
		r := &RateLimiter{client: c}
		return r, nil
	}
}

func (l RateLimiter) GetIsRequestAllowedAndWaitTime(id string) (bool, int, error) {
	ccr, err := l.client.HandleNewRequest(id)
	if err != nil {
		return false, 0, fmt.Errorf("could'nt get number of requests made this interval because %v", err)
	}
	if !ccr.Allowed {
		return false, 0, nil
	}
	return true, 0, nil
}

func (l RateLimiter) HandleRequestsByIP(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//get user's IP
		ip := utils.GetIP(r)
		fmt.Printf("handling request for IP %v\n", ip)
		var ccr cache.CacheClientResponse
		var err error
		if ccr, err = l.client.HandleNewRequest(ip); err != nil {
			fmt.Printf("%+v", err.Error())
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return

		} else if !ccr.Allowed {
			http.Error(w, fmt.Sprintf("Rate limit exceeded. Try again in %v seconds", ccr.WaitFor), http.StatusTooManyRequests)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
