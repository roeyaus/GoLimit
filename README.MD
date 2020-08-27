Hi Airtasker,

Thanks for letting me take part in the challenge.

Implementation is such that a requester's IP will be limited to N requests per X seconds
For our default implementation, We'll use a sliding window algorithm using redis for in-memory persistance. Redis can be later deployed on its own Kubernetes pod or used as a managed cloud service.

** Since the number of requests per time period is configurable, it's necessary to have some constraints with regards to complexity of implementation that also affect efficiency. 
Effectively, if X < 60, we treat the interval verbatim.
if x > 60 but < 3600 , we round down to the nearest minute, and if x > 3600 we round down to the nearest hour.


  1) for each incoming request:
    * get current timestamp.
    * get requester ID
    * round timestamp down to the nearest second/minute/hour depending on X (see constraint above)
    

In addition, we shall provide an interface that can be extended with a different rate limiting implementation and used instead.


Things left to do : 

* make shit work (fix redis go pipeline) <--- DONE
* Make redismock v7.0.1 work <--- DONE
* tests, integration tests <--- DONE
* Add generic extensible interface <-- partially done
* Time to wait <--- DONE, but statistically not accurate.