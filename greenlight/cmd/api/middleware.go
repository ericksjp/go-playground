package main

import (
	"errors"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ericksjp703/greenlight/internal/data"
	"github.com/ericksjp703/greenlight/internal/validator"
	"github.com/felixge/httpsnoop"
	"golang.org/x/time/rate"
)

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// will be acessed via closure :)
	var (
		mu      sync.Mutex
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

func (app *application) recoverPanic(next http.Handler) http.Handler {
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

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this indicates to any cache that the response may vary depending on the Authorization header
		w.Header().Add("Vary", "Authorization")

		// retrieving the Authorization header
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.requestContextWithUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// [Bearer, 'XXXXXXXXXXXXXXX']
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()
		data.ValidateTokenPlaintext(v, token)
		if !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// if no user is found with the given token send a invalidAuthTokenResponse
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			if errors.Is(err, data.ErrRecordNotFound) {
				app.invalidAuthenticationTokenResponse(w, r)
				return
			}
			app.serverErrorResponse(w, r, err)
			return
		}

		r = app.requestContextWithUser(r, user)

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.userFromRequestContext(r)

		if user.IsAnonymous() {
			app.authenticationRequired(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.userFromRequestContext(r)

		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	// we shouldn’t be checking if a user is activated unless we know exactly who they are,
	// so, the authentication middleware will be called before fn
	return app.requireAuthenticatedUser(fn)
}

func (app *application) requirePermission(permission string, next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.userFromRequestContext(r)

		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// return a 403 if the slice dont include the permission
		if !permissions.Include(permission) {
			app.notPermitedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return app.requireActivatedUser(fn)
}

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// inform the client that the response may vary depending on this headers
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		// get the origin of the request from the header
		origin := r.Header.Get("Origin")

		// only run if theres atleast one trusted origin configured and thee
		// origin header is not empty
		if len(app.config.cors.trustedOrigins) > 0 && origin != "" {
			// reflect the origin back to the client if it is present in the trusted origins
			if slices.Contains(app.config.cors.trustedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)

				// treat the request as a preflight request
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					// inform the client that we accept this http methods and headers
					w.Header().Add("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
					w.Header().Add("Access-Control-Allow-Headers", "Authorization, Content-Type")

					// respond the preflight with a 200
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) metrics(next http.Handler) http.Handler {
	// initialize expvar variables (they are thread safe)
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_send")
	requestsPerSecond := expvar.NewFloat("requests_per_second")
	totalProcessingTimeMicros := expvar.NewInt("total_processing_time_microsseconds")
	averageProcessingTimeMicros := expvar.NewInt("average_processing_time_microsseconds")
	totalResponsesSentByStatus := expvar.NewMap("total_responses_sent_by_status")

	// used to compute the rps
	startTime := time.Now()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// increment the requests received
		totalRequestsReceived.Add(1)

		// wrap the next call into httpsnoop method to get metrics
		metrics := httpsnoop.CaptureMetrics(next, w, r)

		// increment the responses send
		totalResponsesSent.Add(1)

		// gets the request processing time from httpsnoop struct and
		// increment the total processing time
		totalProcessingTimeMicros.Add(metrics.Duration.Microseconds())

		// update the map, incrementing by the status code
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)

		requestCount := totalRequestsReceived.Value()
		if requestCount > 0 {
			averageProcessingTimeMicros.Set(totalProcessingTimeMicros.Value() / requestCount)
		}

		elapsedSeconds := time.Since(startTime).Seconds()
		if elapsedSeconds > 0 {
			requestsPerSecond.Set(float64(requestCount) / elapsedSeconds)
		}
	})
}
