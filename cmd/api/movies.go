package main

import (
	"errors"
	"net/http"

	"github.com/ericksjp703/greenlight/internal/data"
	"github.com/ericksjp703/greenlight/internal/validator"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request)  {
	var input data.MovieInput

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	input.Validate(v);

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	movie := &data.Movie{}
	input.ApplyUpdates(movie)

	err = app.models.Movies.Insert(movie);
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app * application) showMovieHandler(w http.ResponseWriter, r *http.Request)  {
	id, err := app.readIDParam(r);
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
	}

	// // the json is created and written in one single step
	// err := json.NewEncoder(w).Encode(movie)

	err = app.writeJSON(w, 200, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	var input data.MovieInput
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	// allow optional values if the method is patch
	if r.Method == "PATCH" {
		input.Validate(v, "Title", "Genres", "Runtime", "Year")
	} else {
		input.Validate(v)
	}

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	input.ApplyUpdates(movie)

	err = app.models.Movies.Update(movie)
	if err != nil {
		if errors.Is(err, data.ErrEditConflict) {
			app.editConflictResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request)  {
	id, err := app.readIDParam(r);
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Movies.Delete(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

    w.WriteHeader(http.StatusNoContent)
}
