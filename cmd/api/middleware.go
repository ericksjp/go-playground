package main

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)

func (app *application) rateLimit(next http.Handler) http.Handler {
	// Rate limiter with a bucket capacity of 4 tokens and a refill rate of
	// 2 tokens per second.
	limiter := rate.NewLimiter(2, 4)

	// use closure to acess the limiter
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// send error if theres no tokens in the bucket
		if !limiter.Allow() {
			app.rateLimitExceededResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

}

func (app *application) recoverPanic(next http.Handler) http.Handler  {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a defered function will always run in case of early exit (panic)
		// in the stack
		defer func() {
			// the recover function check if theres a panic or not
			if err := recover(); err != nil {
				// this header will make go's http server close the connection
				// after a response has been send
				w.Header().Set("Connection", "close")
				// this will log the error using our custom logger and send to
				// the client a status 500
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
