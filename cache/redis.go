package cache

import (
	"context"
	"errors"
	"fmt"
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

func (c RedisClient) RequestAllowedForID(id string) (bool, error) {
	//get timestamp

	currentTS := time.Now().Unix()
	ts := fmt.Sprintf("%d", GetTimestampByInterval(c.intervalInSeconds, currentTS))
	res := c.redisClient.HIncrBy(ctx, id, ts, 1)
	_, err := res.Result()
	if err != nil {
		return false, fmt.Errorf("IncrBy operation failed because : %v", err)
	}
	//we created the key now we must expire it
	return true, c.ExpireUserIdByInterval(id)
}

func GetTimestampByInterval(interval int, ts int64) int64 {
	if interval < 60 {
		return ts
	} else if interval < 3600 {
		return int64(ts/60) * 60
	} else {
		return int64(ts / 3600 * 3600)
	}
}

// ExpireUserIdByInterval sets the user ID to expire after the limit interval defined.
// since we round down to second/minute/hour there won't be that many keys in it.
func (c RedisClient) ExpireUserIdByInterval(id string) error {
	ttl := c.redisClient.TTL(ctx, id)
	if ttl.Err() != nil {
		return fmt.Errorf("TTL operation failed because : %v", ttl.Err())
	}
	if ttl.Val() == -1 {
		cmd := c.redisClient.Expire(ctx, id, time.Second*time.Duration(c.intervalInSeconds))
		if cmd.Err() != nil {
			return fmt.Errorf("Expire operation failed because : %v", cmd.Err())
		}
	}

	return nil
}
