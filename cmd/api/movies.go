package main

import (
	"net/http"
	"time"

	"github.com/ericksjp703/greenlight/internal/data"
)

// func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request)  {}

func (app * application) showMovieHandler(w http.ResponseWriter, _ *http.Request)  {
	movie := data.Movie{
		ID: 1,
		CreatedAt: time.Now(),
		Title: "Inception",
		Year: 2010,
		Runtime: 148,
		Genres: []string{"Action", "Sci-Fi"},
		Version: 1,
	}

	// // the json is created and written in one single step
	// err := json.NewEncoder(w).Encode(movie)

	js, err := movie.MarshalJSON()

	if err != nil {
		http.Error(w, "The server could not process your request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}
