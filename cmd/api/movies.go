package main

import (
	"net/http"
	"time"

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

	// the check function will preserve the first error.
	// In the case of title, if both checks are true, "must be provided" will
	// be on the map
	v.Check(movie.Title == "", "title", "must be provided")
	v.Check(len(movie.Title) > 500, "title", "must not be more than 500 characters long")

	v.Check(movie.Year == 0, "year", "must be provided")
	v.Check(movie.Year < 1988, "year", "must be greater than 1988")
	v.Check(movie.Year > int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime == 0, "runtime", "must be provided")
	v.Check(movie.Runtime < 1, "runtime", "must be a positive integer")

	v.Check(movie.Genres == nil, "genres", "must be provided")
	v.Check(len(movie.Genres) == 0, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) > 5, "genres", "must not be more than 5 genres")
	v.Check(!validator.Unique(movie.Genres), "genres", "must not contain duplicate genres")
	v.Check(validator.In("anime", movie.Genres), "genres", "we dont accept animes, thank you")

	v.Check(movie.Version == 0, "version", "must be provided")
	v.Check(movie.Version < 0, "version", "must be a positive integer")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

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

	movie := data.Movie{
		ID: id,
		CreatedAt: time.Now(),
		Title: "Inception",
		Year: 2010,
		Runtime: 148,
		Genres: []string{"Action", "Sci-Fi"},
		Version: 1,
	}

	// // the json is created and written in one single step
	// err := json.NewEncoder(w).Encode(movie)

	err = app.writeJSON(w, 200, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
