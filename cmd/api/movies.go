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

func (app *application) showMoviesHandler(w http.ResponseWriter, r *http.Request)  {
	movies := data.GetMoviesStore().GetAllMovies()

	err := app.writeJSON(w, http.StatusOK, envelope{"movies": movies}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}


func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request)  {
	id, err := app.readIDParam(r);
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	var movie data.Movie
	err = app.readJSON(w, r, &movie)
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

	err = data.GetMoviesStore().UpdateMovie(id, &movie)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie.ID = id

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

	err = data.GetMoviesStore().DeleteMovie(id)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

    w.WriteHeader(http.StatusNoContent)
}
