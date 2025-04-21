package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
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

func (app *application) readIDParam(r *http.Request) (int64, error) {
	// get the params from the request context (the context is a way to provide data across the requests)
	params := httprouter.ParamsFromContext(r.Context())

	// get the id from the params and try to convert it to int64
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)

	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid id parameter")
	}

	return id, nil
}
