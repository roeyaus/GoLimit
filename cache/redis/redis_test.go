package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/elliotchance/redismock/v7"
	"github.com/go-redis/redis/v7"
	"github.com/stretchr/testify/assert"
)

func getMockRedisClient(intervalInSeconds int, requestsPerInterval int) (*RedisClient, error) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	mc := redismock.NewNiceMock(client)

	return &RedisClient{redisClient: mc, intervalInSeconds: intervalInSeconds, requestsPerInterval: requestsPerInterval}, nil
}
func TestGetTimestampByInterval(t *testing.T) {
	var ts int64
	ts = 1488728901
	newts := getTimestampByInterval(1, int64(ts))
	assert.Equal(t, ts, newts)

	ts = 1488728880
	newts = getTimestampByInterval(65, int64(ts))
	assert.Equal(t, ts, newts)

}

func TestIncAndGetRequestsWithinIntervalSync(t *testing.T) {
	client, _ := getMockRedisClient(1, 1)
	//test multiple requests, same IP, per interval
	req, err := client.IncAndGetRequestsWithinInterval("192.168.0.1", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req)
	}
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.1", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 2, req)
	}
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.1", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 3, req)
	}
	//test one request, same IP, per interval
	client, _ = getMockRedisClient(1, 1)
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.1", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req)
	}
	time.Sleep(2 * time.Second)
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.1", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req)
	}
	time.Sleep(2 * time.Second)

	//test multiple requests, different IPs, per interval
	client, _ = getMockRedisClient(1, 1)
	//test multiple requests, same IP, per interval
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.1", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req)
	}
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.2", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req)
	}
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req)
	}
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 2, req)
	}
}

func TestSlidingWindowConcurrent(t *testing.T) {
	client, _ := getMockRedisClient(1, 1)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(c *RedisClient) {
			_, err := client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
			assert.NoError(t, err)
			wg.Done()
		}(client)
	}
	wg.Wait()
	req, err := client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
	assert.NoError(t, err)
	assert.Equal(t, 21, req)

}

func TestSlidingWindowMove(t *testing.T) {
	client, _ := getMockRedisClient(5, 5)
	for i := 0; i < 4; i++ {
		_, err := client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
		assert.NoError(t, err)
		time.Sleep(1 * time.Second)
	}
	req, err := client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 5, req)
	}
	time.Sleep(5 * time.Second)
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req)
	}
}

//TestIncAndGetRequestsWithinSmallIntervalEdgesConcurrent tests concurrent requests right near the end of an interval
//and right at the beginning of the next one
func TestSlidingWindowEdgesConcurrent(t *testing.T) {
	client, _ := getMockRedisClient(5, 5)
	time.Sleep(4 * time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(c *RedisClient) {
			_, err := client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
			assert.NoError(t, err)
			wg.Done()
		}(client)
	}
	wg.Wait()
	req, err := client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 6, req)
	}
	time.Sleep(2 * time.Second)
	req, err = client.IncAndGetRequestsWithinInterval("192.168.0.3", client.intervalInSeconds)
	if assert.NoError(t, err) {
		assert.Equal(t, 7, req)
	}
}
