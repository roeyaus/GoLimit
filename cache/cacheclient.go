package cache

type CacheClient interface {
	HandleNewRequest(id string) (CacheClientResponse, error)
}

type CacheClientResponse struct {
	Allowed              bool
	WaitFor              int
	RequestsMadeInWindow int
}
