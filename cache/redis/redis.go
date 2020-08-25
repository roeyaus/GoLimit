package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type RedisClient struct {
	redisClient         *redis.Client
	intervalInSeconds   int
	requestsPerInterval int
	numSecondsForCheck  int
}

func GetRedisClient(intervalInSeconds int, requestsPerInterval int) (*RedisClient, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	if c == nil {
		return nil, errors.New("cannot create redis client")
	}

	return &RedisClient{redisClient: c, intervalInSeconds: intervalInSeconds, requestsPerInterval: requestsPerInterval}, nil
}

func (c RedisClient) GetRequestsWithinInterval(id string, intervalInSeconds int) (int, error) {
	//get timestamp
	roundedTS := getTimestampByInterval(intervalInSeconds, time.Now().Unix())
	roundedts := fmt.Sprintf("%d", roundedTS)
	historicTS := roundedTS - int64(intervalInSeconds)
	//get all timestamps for this ID
	fmt.Printf("%v %v\n", roundedTS, historicTS)

	//first we increment requests for rounded ts and get all the timestamps
	var cmd2 *redis.StringStringMapCmd
	pipe := c.redisClient.TxPipeline()
	_ = pipe.HIncrBy(ctx, id, roundedts, 1)

	cmd2 = pipe.HGetAll(ctx, id)
	_, err := pipe.Exec(ctx)
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
	totalRequestsInInterval := 0
	for k := range list {
		n, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("can't parse timestamp string %v", k)
		}
		if n > historicTS {
			numRequests, err := strconv.ParseInt(list[k], 10, 32)
			if err != nil {
				return 0, fmt.Errorf("can't parse timestamp string %v", k)
			}
			totalRequestsInInterval += int(numRequests)
		}

	}

	//now check if the key is set to expire or not and create an expiry for it
	ttl := c.redisClient.TTL(ctx, id)
	if ttl.Err() != nil {
		return 0, fmt.Errorf("TTL operation failed because : %v", ttl.Err())
	}
	if ttl.Val() == -1 {
		_ = c.redisClient.Expire(ctx, id, time.Second*time.Duration(intervalInSeconds))
	}
	if err != nil {
		return 0, fmt.Errorf("Pipeline operation failed because : %v", err)
	}

	//we created the key now we must expire it
	return totalRequestsInInterval, err
}

func getTimestampByInterval(intervalInSeconds int, ts int64) int64 {
	if intervalInSeconds < 60 {
		return ts
	} else if intervalInSeconds < 3600 {
		return int64(ts/60) * 60
	} else {
		return int64(ts/3600) * 3600
	}
}
