Hi Airtasker,

Thanks for letting me take part in the challenge.

RateLimiter is a Golang package for rate-limiting requests. 

Usage : 

```
limiter, err := ratelimiter.NewRateLimiter(WindowSize, RequestsPerWindow)
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


Implementation is such that a requester's IP will be limited to N requests per X seconds
For our default implementation, We'll use a sliding window algorithm using redis for in-memory persistance. Redis can be later deployed on its own Kubernetes pod or used as a managed cloud service.

** Since the number of requests per time period is configurable, it's necessary to have some constraints to save memory and processing time. 
Effectively, if X < 60, we treat the interval verbatim. (in seconds)
if x > 60 but <= 3600 , we round timestamps down to the nearest minute, and if x > 3600 we round down to the nearest hour.
This is because if we had a window of 10 hours and we measure every second we would have 36000 entries per user at any given time. By having less resolution on our request counts we limit that to basically ~60 entries per user at the most.

As mentioned, The algorithm is a sliding window with a small twist .
Since our resolution is sometimes coarse, for larger intervals we treat the requests made in the current window as distributed *evenly*.  (See this document : https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)

*** Doing it like that means it's hard to measure the time left until the user can make a request and indeed my implementation doesn't do a very good job. I couldn't think of a better one within a reasonable period of time. *** 

We'll use Redis to handle the request counts etc.
This is because :
* We want a distributed system able to maintain rate limiting for more than one server.
* We don't want to slow down request handling times too much.

With Redis, we use a Hash and save all previous request timestamps within our defined window. We do this in different resolutions (sec/min/hour) depending on window. See above.




    

In addition, we shall provide an interface that can be extended with a different rate limiting implementation and used instead.


Things left to do : 

* make shit work (fix redis go pipeline) <--- DONE
* Make redismock v7.0.1 work <--- DONE
* tests, integration tests <--- DONE
* Add generic extensible interface <-- partially done
* Time to wait <--- DONE, but statistically not accurate.