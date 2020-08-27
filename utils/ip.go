package utils

import (
	"net/http"
)

// GetIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
func GetIP(r *http.Request) string {
	//this seems to be an NGINX header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	//this is generally what we use to identify original IP from behind a proxy/LB
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	//this will also return the port in localhost
	return r.RemoteAddr
}
