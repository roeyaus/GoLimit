package cache

type CacheClient interface {
	IncAndGetRequestsWithinInterval(id string, intervalInSeconds int) (int, error)
}
