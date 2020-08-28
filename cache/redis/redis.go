package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/roeyaus/airtasker/cache"
)

var ctx = context.Background()

type RedisClient struct {
	redisClient          redis.Cmdable
	WindowInSeconds      int
	MaxRequestsPerWindow int
}

func GetRedisClient(redisAddr string, redisPassword string, redisDB int, windowInSeconds int, maxRequestsPerWindow int) (*RedisClient, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword, // no password set
		DB:       redisDB,       // use default DB
	})
	if c == nil {
		return nil, errors.New("cannot create redis client")
	}

	return &RedisClient{redisClient: c, WindowInSeconds: windowInSeconds, MaxRequestsPerWindow: maxRequestsPerWindow}, nil
}

//HandleNewRequest will atomically increase request count for id by 1 and return the updated number of requests
//made for that id within the interval, if the request is allowed and how long to wait until it is
func (c RedisClient) HandleNewRequest(id string) (cache.CacheClientResponse, error) {
	resp := cache.CacheClientResponse{Allowed: false, WaitFor: 0}
	currentTS := time.Now().Unix()
	windowInSeconds := c.WindowInSeconds
	//get timestamp
	roundedTS := getTimestampByInterval(windowInSeconds, currentTS)
	roundedts := fmt.Sprintf("%d", roundedTS)
	historicTS := roundedTS - int64(windowInSeconds)
	//get all timestamps for this ID
	fmt.Printf("%v %v\n", roundedTS, historicTS)

	//first we increment requests for rounded ts and get all the timestamps atomically
	//even if multiple clients arrive here at the same time, they will each see a valid request count
	//if the key(userID) expires this just recreates it with the current timestamp
	var cmd2 *redis.StringStringMapCmd
	pipe := c.redisClient.TxPipeline()
	_ = pipe.HIncrBy(id, roundedts, 1)

	cmd2 = pipe.HGetAll(id)
	_, err := pipe.Exec()
	if err != nil {
		return resp, fmt.Errorf("Pipeline operation failed because : %v", err)
	}

	list, err := cmd2.Result()
	if err != nil {
		return resp, fmt.Errorf("HGetAll operation failed because : %v", err)
	}

	//then we can do all the other actions at our leisure, because if we've already exceeded the request count
	//we'll get denied anyway.
	//We don't mind going through all the timestamps everytime because our constraint guarantees there won't be that many.
	totalRequestsInPrevInterval := 0
	totalRequestsInInterval := 0
	numPrevIntervals := 0
	minWindowTS := currentTS
	for k := range list {
		n, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			return resp, fmt.Errorf("can't parse timestamp string %v", k)
		}
		if n >= historicTS {
			if n < minWindowTS {
				minWindowTS = n

			}
			numRequests, err := strconv.ParseInt(list[k], 10, 32)
			fmt.Printf("ts : %v, req: %v\n", n, numRequests)
			if err != nil {
				return resp, fmt.Errorf("can't parse timestamp string %v", k)
			}
			if n < roundedTS {
				totalRequestsInPrevInterval += int(numRequests)
				numPrevIntervals++
			} else {
				totalRequestsInInterval += int(numRequests)
			}

		}

	}

	//get seconds elapsed since start of interval
	//an interval is either 1 second, one minute or one hour
	secondsElapsed := int(currentTS - roundedTS)
	fmt.Printf("totalRequestsInPrevInterval : %v, totalRequestsInInterval : %v\n", totalRequestsInPrevInterval, totalRequestsInInterval)
	fmt.Printf("secondsElapsedInInterval : %v, minWindowTS : %v\n", secondsElapsed, minWindowTS)
	if currentTS != minWindowTS && int(currentTS-minWindowTS) < windowInSeconds {
		windowInSeconds = int(currentTS - minWindowTS)
	}
	//we know the number of requests made during the previous interval,
	//now we use this calculation :
	//requests in prev interval * ((window - seconds elapsed in current interval)/window) + requests in current interval
	//This spaces out requests evenly across the window, so some precision is lost.
	totalRequestsInWindow := int(float32(totalRequestsInPrevInterval)*(float32(c.WindowInSeconds-secondsElapsed)/float32(c.WindowInSeconds))) + totalRequestsInInterval

	//now check if the key is set to expire or not and create an expiry for it
	_ = c.redisClient.Expire(id, time.Second*time.Duration(c.WindowInSeconds))

	resp.Allowed = totalRequestsInWindow < c.MaxRequestsPerWindow
	if !resp.Allowed {
		resp.WaitFor = c.calculateTimeToWait(totalRequestsInWindow)
	}
	resp.RequestsMadeInWindow = totalRequestsInWindow
	fmt.Printf("%v", resp)

	return resp, err
}

func getTimestampByInterval(windowInSeconds int, ts int64) int64 {
	if windowInSeconds <= 60 {
		return ts
	} else if windowInSeconds <= 3600 {
		return int64(ts/60) * 60
	} else {
		return int64(ts/3600) * 3600
	}
}

//we calculate the time to wait until we can handle another request
//statistically space out the requests in the window evenly
//**** this is not accurate due to the fact that it's calculated much the same way as the sliding window.
func (c RedisClient) calculateTimeToWait(totalRequestsInWindow int) int {

	secBetweenRequests := int(float32(c.WindowInSeconds) / float32(totalRequestsInWindow))
	fmt.Printf("WindowInSeconds : %v, totalRequestsInWindow : %v, secBetweenRequests : %v\n", c.WindowInSeconds, totalRequestsInWindow, secBetweenRequests)
	return (totalRequestsInWindow - c.MaxRequestsPerWindow + 1) * secBetweenRequests
}
