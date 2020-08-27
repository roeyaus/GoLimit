package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	ratelimiter "github.com/roeyaus/airtasker/ratelimiter"
	"github.com/roeyaus/airtasker/utils"
)

func main() {
	limiter, err := ratelimiter.NewRateLimiter(30, 5)
	if err != nil {
		fmt.Printf("%v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", ExampleHandler)
	if err := http.ListenAndServe(":8080", limiter.HandleRequestsByIP(mux)); err != nil {
		panic(err)
	}
}

func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]string{
		"ip": utils.GetIP(r),
	})
	w.Write(resp)
}
