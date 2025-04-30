package main

import (
	"context"
	"net/http"

	"github.com/ericksjp703/greenlight/internal/data"
)

// With both the type and the typed constant of the key being unexported, no code
// from outside your package can put data into the context that would cause a collision.

type contextKey uint8

const (
	_ contextKey = iota
	userKey
	// tokenKey
)

// updates the request context with the given user
func (app *application) requestContextWithUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userKey, user)
	return r.WithContext(ctx)
}

// gets a user from a request context
func (app *application) userFromRequestContext(r *http.Request) *data.User {
	val, ok := r.Context().Value(userKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}
	return val
}
