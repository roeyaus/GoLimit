package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
)

var ctx = context.Background()

type RedisClient struct {
	redisClient         redis.Cmdable
	intervalInSeconds   int
	requestsPerInterval int
	numSecondsForCheck  int
}

func GetRedisClient(redisAddr string, redisPassword string, redisDB int, intervalInSeconds int, requestsPerInterval int) (*RedisClient, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword, // no password set
		DB:       redisDB,       // use default DB
	})
	if c == nil {
		return nil, errors.New("cannot create redis client")
	}

	return &RedisClient{redisClient: c, intervalInSeconds: intervalInSeconds, requestsPerInterval: requestsPerInterval}, nil
}

func (c RedisClient) IncAndGetRequestsWithinInterval(id string, intervalInSeconds int) (int, error) {
	//get timestamp
	roundedTS := getTimestampByInterval(intervalInSeconds, time.Now().Unix())
	roundedts := fmt.Sprintf("%d", roundedTS)
	historicTS := roundedTS - int64(intervalInSeconds)
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
		return 0, fmt.Errorf("Pipeline operation failed because : %v", err)
	}

	list, err := cmd2.Result()
	if err != nil {
		return 0, fmt.Errorf("HGetAll operation failed because : %v", err)
	}

	//then we can do all the other actions at our leisure, because if we've already exceeded the request count
	//we'll get denied anyway.
	//We don't mind going through all the timestamps everytime because our constraint guarantees there won't be that many.
	totalRequestsInPrevInterval := 0
	totalRequestsInInterval := 0
	for k := range list {
		n, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("can't parse timestamp string %v", k)
		}
		if n > historicTS {
			numRequests, err := strconv.ParseInt(list[k], 10, 32)
			fmt.Printf("ts : %v, req: %v\n", n, numRequests)
			if err != nil {
				return 0, fmt.Errorf("can't parse timestamp string %v", k)
			}
			if n < roundedTS {
				totalRequestsInPrevInterval += int(numRequests)
			} else {
				totalRequestsInInterval += int(numRequests)
			}

		}

	}

	//get seconds elapsed since start of interval
	secondsElapsed := int(time.Now().Unix() - roundedTS)
	fmt.Printf("totalRequestsInPrevInterval : %v, totalRequestsInInterval : %v\n", totalRequestsInPrevInterval, totalRequestsInInterval)
	fmt.Printf("secondsElapsed : %v, ", secondsElapsed)
	totalRequestsInWindow := int(float32(totalRequestsInPrevInterval)*(float32(intervalInSeconds-secondsElapsed)/float32(intervalInSeconds))) + totalRequestsInInterval
	fmt.Printf("totalRequestsInWindow : %v\n", totalRequestsInWindow)
	//now check if the key is set to expire or not and create an expiry for it
	_ = c.redisClient.Expire(id, time.Second*time.Duration(intervalInSeconds))
	//we know the number of requests made during the previous interval,
	//now we use this calculation : 42 * ((60-15)/60) + 18 to estimate the number of requests within our window

	return totalRequestsInWindow, err
}

func getTimestampByInterval(intervalInSeconds int, ts int64) int64 {
	if intervalInSeconds <= 60 {
		return ts
	} else if intervalInSeconds < 3600 {
		return int64(ts/60) * 60
	} else {
		return int64(ts/3600) * 3600
	}
}
