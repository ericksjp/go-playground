package main

import (
	"net/http"
	"time"

	"github.com/ericksjp703/greenlight/internal/data"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request)  {
	var movie data.Movie

	err := app.readJSON(w, r, &movie)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = app.WriteJSON(w, http.StatusCreated, envelope{"movie": movie}, nil)
	if err != nil {
		http.Error(w, "The server could not process your request", http.StatusInternalServerError)
		return
	}
}

func (app * application) showMovieHandler(w http.ResponseWriter, r *http.Request)  {
	// get the id from the request
	id, err := app.readIDParam(r);
	if err != nil {
		http.NotFound(w, r)
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

	err = app.WriteJSON(w, 200, envelope{"movie": movie}, nil)
	if err != nil {
		http.Error(w, "The server could not process your request", http.StatusInternalServerError)
		return
	}
}
