package main

import (
	"net/http"

	"github.com/ericksjp703/greenlight/internal/data"
	"github.com/ericksjp703/greenlight/internal/validator"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request)  {
	var movie data.Movie

	err := app.readJSON(w, r, &movie)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	movie.Validate(v);

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	movie = data.GetMoviesStore().AddMovie(movie)

	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app * application) showMovieHandler(w http.ResponseWriter, r *http.Request)  {
	// get the id from the request
	id, err := app.readIDParam(r);
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := data.GetMoviesStore().GetMovie(id)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// // the json is created and written in one single step
	// err := json.NewEncoder(w).Encode(movie)

	err = app.writeJSON(w, 200, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
