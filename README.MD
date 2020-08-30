Hi Airtasker,

Thanks for letting me take part in the challenge.

RateLimiter is a Golang package for rate-limiting requests. 

**Usage :** 

```
import "github.com/roeyaus/airtasker/ratelimiter"

//use default cache client (redis + sliding window)
limiter, err := ratelimiter.NewRateLimiter(WindowSize, RequestsPerWindow)

//or to use a different strategy/client
cacheClient = &MyCacheClient{}
limiter, err := ratelimiter.NewRateLimiterWithCacheClient(WindowSize, RequestsPerWindow, cacheClient)

if err != nil {
  fmt.Printf("%v", err)
}

//to handle requests directly by IP
mux := http.NewServeMux()
mux.HandleFunc("/", ExampleHandler)
if err := http.ListenAndServe(":8080", limiter.HandleRequestsByIP(mux)); err != nil {
  panic(err)
}
//for other ID's that are not IPs, or to just use rate limiting : 
allowed, waitFor, err := limiter.GetIsRequestAllowedAndWaitTime(<UNIQUE ID>) //rate limits anything by unique ID, returns if request is allowed and time to wait until allowed if not 
```


**For running with real redis, run :** 
```
docker run --name some-redis -p 6379:6379 -d redis redis-server
```

which will create a local redis container. Just remember to instantiate the RateLimiter with redisHost="localhost:6379" , redisPasword="" and redisDB=0 

**Implementation:**

Implementation is such that a requester's IP will be limited to N requests per X seconds
For our default implementation, We'll use a sliding window algorithm using redis for in-memory persistance. Redis can be later deployed on its own Kubernetes pod or used as a managed cloud service.

** Since the number of requests per time period is configurable, it's necessary to have some constraints to save memory and processing time. 

Effectively, if X < 60, we treat the interval verbatim. (in seconds)
if x > 60 but <= 3600 , we round timestamps down to the nearest minute, and if x > 3600 we round down to the nearest hour.

This is because if we had a window of 10 hours and we measure every second we would have 36000 entries per user at any given time. By having less resolution on our request counts we limit that to basically ~60 entries per user at the most.

As mentioned, The algorithm is a sliding window with a small twist .
Since our resolution is sometimes coarse, for larger intervals we treat the requests made in the current window as distributed *evenly*.  (See this document : https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)

**Doing it like that means it's hard to measure the time left until the user can make a request and indeed my implementation doesn't do a very good job. I couldn't think of a better one within a reasonable period of time.**

We'll use Redis to handle the request counts etc.
This is because :
* We want a distributed system able to maintain rate limiting for more than one server.
* We don't want to slow down request handling times too much.

With Redis, we use a Hash and save all previous request timestamps within our defined window. We do this in different resolutions (sec/min/hour) depending on window. See above.


**Extensibility:**

ratelimiter provides a CacheClient interface to implement a different rate limiting strategy using a different kind of caching mechanism.

```
type CacheClient interface {
	HandleNewRequest(id string) (CacheClientResponse, error)
}

type CacheClientResponse struct {
	Allowed              bool
	WaitFor              int
	RequestsMadeInWindow int
}
```

Then instantiate a new ratelimiter with your customClient 
```
limiter, err := ratelimiter.NewRateLimiterWithCacheClient(WindowSize, RequestsPerWindow, customClient)
```

**Testing:**

I use a Redis mocking library for testing in order to alleviate the need to create a local redis cluster.
The tests cover most of the basic necessities of the redis cache implementation, and some general RateLimiter tests as well.



**Airtasker values**

Stay open  - for me this means that things should not be allowed to simmer in the background. If something's not working like it should, there should be a (friendly) discussion on it as soon as possible and expectations should be set well, both for projects and people. 

Fit for purpose - I'm a full-on pragmatist. I will use whatever tools are best for the job, not just because they're new or "trendy". I treat anything I do in this way.

People matter - completely agree, I've been in too many companies who treat people like "resources". I'd say that someone who feels the company is looking out for them is at least 2x more productive.

When it’s on it’s on - As someone's whose been doing startups for the past few years, I know how intense it can be when something's urgent. I've also learned not to over-do "urgent" since that blurs the lines between what's actually important and what's not.

Own it - I believe in complete ownership. Everyone should feel like they own their work - that's when people really start contributing. 

**Lastly... about me** 

● What is important to you in your career? 
Past couple of years I've only been doing startups which a different kind of challenge.
I'm currently looking to work where I can see actual scale from millions of users.

● What excites you about Airtasker and the opportunity? 
I think Airtasker is well positioned in the global gig economy and the scale things are at seems really interesting. Also seems to be alot of opportunities to contribute in different parts of the business.

● What would you like us to know about you? 
I've been playing guitar drums and bass guitar for more than 20 years and am an avid music lover.
