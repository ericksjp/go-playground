
package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter *rate.Limiter
		lastSeen time.Time
	}

	// will be acessed via closure :)
	var (
		mu sync.Mutex
		clients = make(map[string]*client) // key = ip
	)

	// go routine that clean the clients map once every second
	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()

			now := time.Now()
			deadline := time.Minute * 3

			// Iterate through all ips and remove any ip from the map
			// if they have not accessed the server within the deadline
			for ip, client := range clients {
				if now.Sub(client.lastSeen) > deadline {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// extracting the client ip
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// locking access to the clients map. 
		// using defer to unlock is avoided here because it would delay unlocking 
		// until all middlewares in the chain have completed. this could slow down 
		// server access if there is a heavy operation in the chain
		mu.Lock()

		// creating one limiter for the ip if its not in the map
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
		}

		// update last seen time for the ip
		clients[ip].lastSeen = time.Now()

		// sending error if there is no tokens in the bucket for the ip
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		mu.Unlock()

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
