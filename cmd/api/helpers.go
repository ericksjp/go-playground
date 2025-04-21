package main

import (
	"encoding/json"
	"net/http"
	"maps"
)

// this will add additional layer of abstraction to the json response
// "movie": {...}
type envelope map[string]any

func (app *application) WriteJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// encode the data to json, return error
	js ,err := json.Marshal(data)
	if err != nil {
		return err
	}

	// copy the headers to the response writer
	maps.Copy(w.Header(), headers)

	// set the content type, status code and write the json to the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

