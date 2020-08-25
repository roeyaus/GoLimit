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

func (c RedisClient) GetRequestsWithinInterval(id string, intervalInSeconds int) (int, error) {
	//get timestamp
	roundedTS := GetTimestampByInterval(intervalInSeconds, time.Now().Unix())
	roundedts := fmt.Sprintf("%d", roundedTS)
	historicTS := roundedTS - int64(intervalInSeconds)
	hts := fmt.Sprintf("%d", historicTS)
	//get all timestamps for this ID
	fmt.Printf("%v %v\n", roundedts, hts)
	pipe := c.redisClient.TxPipeline()
	cmd := pipe.ZRangeByScoreWithScores(ctx, id, &redis.ZRangeBy{Min: hts, Max: roundedts, Offset: 0, Count: 0})
	if cmd.Err() != nil {
		return 0, fmt.Errorf("ZRangeByScoreWithScores operation failed because : %v", cmd.Err())
	}
	totalRequestsInInterval := 0
	newNumRequests := 0
	for _, v := range cmd.Val() {
		if numRequests, ok := v.Member.(int); ok {
			if int64(v.Score) == roundedTS {
				newNumRequests = numRequests
			}
			totalRequestsInInterval += numRequests
		} else {
			return 0, fmt.Errorf("Cannot aggregate num requests, %v is not int", v.Member)
		}
	}
	fmt.Printf("numRequests for current : %v,  totalRequestsInInterval : %v\n", newNumRequests, totalRequestsInInterval)
	//remove the current request count in the sorted set for the current id/timestamp
	cmd2 := pipe.ZRemRangeByScore(ctx, id, roundedts, roundedts)
	if cmd2.Err() != nil {
		return 0, fmt.Errorf("ZRemRangeByScore operation failed because : %v", cmd2.Err())
	}
	//increment and push a new one in
	newNumRequests++
	cmd3 := pipe.ZAdd(ctx, id, &redis.Z{Score: float64(roundedTS), Member: newNumRequests})
	if cmd3.Err() != nil {
		return 0, fmt.Errorf("ZAdd operation failed because : %v", cmd3.Err())
	}
	fmt.Printf("%v", cmd3.Val())
	//now check if the key is set to expire or not and create an expiry for it
	ttl := pipe.TTL(ctx, id)
	if ttl.Err() != nil {
		return 0, fmt.Errorf("TTL operation failed because : %v", ttl.Err())
	}
	if ttl.Val() == -1 {
		cmd := pipe.Expire(ctx, id, time.Second*time.Duration(intervalInSeconds))
		if cmd.Err() != nil {
			return 0, fmt.Errorf("Expire operation failed because : %v", cmd.Err())
		}
	}
	_, err := pipe.Exec(ctx)
	// cmd := pipe.HGetAll(ctx, id)

	// list, err := cmd.Result()
	// if err != nil {
	// 	return false, fmt.Errorf("HGetAll operation failed because : %v", err)
	// }
	// keys := make([]int64, 0, len(list))
	// for k := range list {
	// 	n, err := strconv.ParseInt(k, 10, 64)
	// 	if err != nil {
	// 		return false, fmt.Errorf("can't parse timestamp string %v", k)
	// 	}
	// 	keys = append(keys, n)
	// }
	// sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	// i := sort.Search(len(keys), func(i int) bool { return keys[i] >= currentTS-int64(c.intervalInSeconds) })
	// if i >= len(keys) {
	// 	return
	// }

	// res := pipe.HIncrBy(ctx, id, ts, 1)
	// _, err = res.Result()
	// if err != nil {
	// 	return false, fmt.Errorf("IncrBy operation failed because : %v", err)
	// }

	//we created the key now we must expire it
	return totalRequestsInInterval, err
}

func GetTimestampByInterval(intervalInSeconds int, ts int64) int64 {
	if intervalInSeconds < 60 {
		return ts
	} else if intervalInSeconds < 3600 {
		return int64(ts/60) * 60
	} else {
		return int64(ts/3600) * 3600
	}
}
