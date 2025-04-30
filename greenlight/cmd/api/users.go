package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/ericksjp703/greenlight/internal/data"
	"github.com/ericksjp703/greenlight/internal/validator"
)

func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	user, err := app.models.Users.Get(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input data.UserInput

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	input.Validate(v)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	var user data.User
	err = input.UpdateUserFields(&user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user.Activated = false

	err = app.models.Users.Insert(&user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// 3 day expiration token
	token, err := app.models.Token.New(user.ID, time.Hour * 24 * 3, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// launch the logic to send the email in a goroutine
	app.background(func() {

		data := map[string]any {
			"userID": user.ID,
			"activationToken": token.Plaintext,
		}

		err := app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	// the status acepted indicates that the request has been accepted for processing,
	// but the processing has not been completed yet.
	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	var input data.UserInput
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	// allow optional values if the method is patch
	if r.Method == "PATCH" {
		input.Validate(v, "Name", "Email", "Password")
	} else {
		input.Validate(v)
	}

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.Get(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = input.UpdateUserFields(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Users.Delete(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	var input struct {
		Token string `json:"token"`
	}

	err = app.readJSON(w, r, &input);
	if err != nil {
		app.badRequestResponse(w, r, err);
		return
	}

	v := validator.New()
	data.ValidateTokenPlaintext(v, input.Token)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
	}

	// get a user for the given token scope and plaintext
	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.Token)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	if (user.ID != id) {
		app.notFoundResponse(w, r)
		return
	}

	// update the user activation status
	user.Activated = true

	// save the update user on the db
	err = app.models.Users.Update(user)
	if err != nil {
		if errors.Is(err, data.ErrEditConflict) {
			app.editConflictResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	// delete all activation tokens for the user
	err = app.models.Token.DeleteAllForUser(user.ID, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
