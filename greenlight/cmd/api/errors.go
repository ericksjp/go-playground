package main

import (
	"fmt"
	"net/http"
)

// helper to use the app logger to log errors
func (app *application) logError(r *http.Request, err error) {
	// using the logger to include  current request method and url in the log entry
	app.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url": r.URL.String(),
	})
}

// generic error response
// this basically will just envelope the error message with "error" and send
// a status 500 to the client if the error cannot be written to the response
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	err := app.writeJSON(w, status, envelope{"error": message}, nil)

	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

// specific for 500 internal server error
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

// specific for 404 not found
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

// specific for 405 method not allowed
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// specific for 400 bad request
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error)  {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// specific for 422 unprocessable entity
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string)  {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// specific for 409 status conflict
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(w, r, http.StatusConflict, message)
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

// specific for 401 status unauthorized (invalid token)
func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	// inform the client that we want this token
	w.Header().Set("WWW-Authenticate", "Bearer")

	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// specific for 401 status unauthorized (user not authenticated)
func (app *application) authenticationRequired(w http.ResponseWriter, r *http.Request)  {
	message := "you must be authenticate to acess this resource"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// specific for 403 status forbidden
func (app *application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request)  {
	message := "your user account must be activated to acess this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}

// specific for 403 status forbidden
func (app *application) notPermitedResponse(w http.ResponseWriter, r *http.Request)  {
	message := "your user account doesn't have the necessary permissions to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}
