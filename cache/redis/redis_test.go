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

func getMockRedisClient(windowInSeconds int, maxRequestsPerWindow int) (*RedisClient, error) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	mc := redismock.NewNiceMock(client)

	return &RedisClient{redisClient: mc, WindowInSeconds: windowInSeconds, MaxRequestsPerWindow: maxRequestsPerWindow}, nil
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

func TestHandleNewRequestSync(t *testing.T) {
	client, _ := getMockRedisClient(1, 1)
	//test multiple requests, same IP, per interval
	req, err := client.HandleNewRequest("192.168.0.1")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
	}
	req, err = client.HandleNewRequest("192.168.0.1")
	if assert.NoError(t, err) {
		assert.Equal(t, 2, req.RequestsMadeInWindow)
	}
	req, err = client.HandleNewRequest("192.168.0.1")
	if assert.NoError(t, err) {
		assert.Equal(t, 3, req.RequestsMadeInWindow)
	}
	//test one request, same IP, per window
	client, _ = getMockRedisClient(5, 1)
	req, err = client.HandleNewRequest("192.168.0.1")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
	}
	time.Sleep(2 * time.Second)
	req, err = client.HandleNewRequest("192.168.0.1")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
	}
	time.Sleep(2 * time.Second)
	req, err = client.HandleNewRequest("192.168.0.1")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
	}

	//test multiple requests, different IPs, per interval
	client, _ = getMockRedisClient(1, 1)
	//test multiple requests, same IP, per interval
	req, err = client.HandleNewRequest("192.168.0.1")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
	}
	req, err = client.HandleNewRequest("192.168.0.2")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
	}
	req, err = client.HandleNewRequest("192.168.0.3")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
	}
	req, err = client.HandleNewRequest("192.168.0.3")
	if assert.NoError(t, err) {
		assert.Equal(t, 2, req.RequestsMadeInWindow)
	}
}

func TestSlidingWindowConcurrent(t *testing.T) {
	client, _ := getMockRedisClient(5, 1)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(c *RedisClient) {
			_, err := client.HandleNewRequest("192.168.0.3")
			assert.NoError(t, err)
			wg.Done()
		}(client)
	}
	wg.Wait()
	req, err := client.HandleNewRequest("192.168.0.3")
	assert.NoError(t, err)
	assert.Equal(t, 21, req.RequestsMadeInWindow)

}

func TestSlidingWindowMove(t *testing.T) {
	client, _ := getMockRedisClient(5, 4)
	for i := 0; i < 4; i++ {
		_, err := client.HandleNewRequest("192.168.0.3")
		assert.NoError(t, err)
		time.Sleep(1 * time.Second)
	}
	req, err := client.HandleNewRequest("192.168.0.3")
	if assert.NoError(t, err) {
		assert.Equal(t, 5, req.RequestsMadeInWindow)
		assert.Equal(t, false, req.Allowed)
	}
	time.Sleep(6 * time.Second)
	req, err = client.HandleNewRequest("192.168.0.3")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, req.RequestsMadeInWindow)
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
			_, err := client.HandleNewRequest("192.168.0.3")
			assert.NoError(t, err)
			wg.Done()
		}(client)
	}
	wg.Wait()
	req, err := client.HandleNewRequest("192.168.0.3")
	if assert.NoError(t, err) {
		assert.Equal(t, 6, req.RequestsMadeInWindow)
	}
	time.Sleep(2 * time.Second)
	req, err = client.HandleNewRequest("192.168.0.3")
	if assert.NoError(t, err) {
		assert.Equal(t, 7, req.RequestsMadeInWindow)
	}
}

func testCalculateTimeToWait(t *testing.T) {

}
