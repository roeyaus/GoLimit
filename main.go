package main

import (
	"encoding/json"
	"net/http"

	"github.com/roeyaus/airtasker/ratelimiter"
)

func main() {
	limiter := ratelimiter.NewRateLimiter(3600, 100)

	http.HandleFunc("/", limit.HandleRequest(ExampleHandler))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]string{
		"ip": GetIP(r),
	})
	w.Write(resp)
}
