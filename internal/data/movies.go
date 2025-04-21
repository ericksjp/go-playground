package data

import (
	"encoding/json"
	"fmt"
	"time"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   int32     `json:"-"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

// this is a custom MarshalJSON method for the Movie struct.
// It allows us to customize the JSON output for the Movie struct.
func (m Movie) MarshalJSON() ([]byte, error) {
	var runtime string;
	if m.Runtime != 0 {
		runtime = fmt.Sprintf("%d minutes", m.Runtime)
	}

	type MovieAlias Movie

	// Create an anonymous struct that includes the fields we want to include
	// in the JSON output. override "Runtime" field to be a string.
	out := struct {
		MovieAlias
		Runtime string `json:"runtime,omitempty"`
	} {
		MovieAlias: MovieAlias(m),
		Runtime: runtime,
	}

	return json.Marshal(out)
}
