package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ericksjp703/greenlight/internal/validator"
	"github.com/julienschmidt/httprouter"
)

// this will add additional layer of abstraction to the json response
// "movie": {...}
type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
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

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// limit the size of the request body to prevent malicious large payloads.
	const maxBytes = 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// create a json decoder and disallow unknown fields.
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	// decode the json into the destination.
	err := decoder.Decode(dst);
	if err != nil {
		return handleJSONDecodeError(err)
	}

	// ensure the request body contains only a single json value
	err = decoder.Decode(&struct{}{}); 
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// processes errors returned by the json decoder
func handleJSONDecodeError(err error) error {
	var (
		syntaxError           *json.SyntaxError
		unmarshalTypeError    *json.UnmarshalTypeError
		invalidUnmarshalError *json.InvalidUnmarshalError
		maxBytesError         *http.MaxBytesError
	)

	switch {
	// handle syntax errors in the json
	case errors.As(err, &syntaxError):
		return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

	// handle type error in the json
	case errors.As(err, &unmarshalTypeError):
		if unmarshalTypeError.Field != "" {
			return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
		}
		return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

	// handle request body too large
	case errors.As(err, &maxBytesError):
		return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

	// handle invalid unmarshal errors (stop, there is a bug sir)
	case errors.As(err, &invalidUnmarshalError):
		panic(err)

	// handle unknown fields in the json
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return fmt.Errorf("body contains unknown key %s", fieldName)

	// handle empty body errors.
	case errors.Is(err, io.EOF):
		return errors.New("body must not be empty")

	// handle unexpected EOF errors
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("body contains badly-formed JSON")
	}

	// return the original error if no specific case matches
	return err
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

// read key from the query string and return the value or default value
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

// read key from query where there are multiple values separated by comma
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)
	if csv == "" {
		return defaultValue
	}
	return strings.Split(csv, ",")
}

// read key from query and transform to integer
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

// executes a function in a separate goroutine and recovers from panic
func (app *application) background(fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintFatal(fmt.Errorf("%s", err), nil);
			}
		}()

		fn()
	}()
}
