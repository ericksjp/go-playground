package data

import (
	"time"

	"github.com/ericksjp703/greenlight/internal/validator"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

func (m Movie) Validate(v *validator.Validator)  {

	// the check function will preserve the first error.
	// In the case of title, if both checks are true, "must be provided" will
	// be on the map
	v.Check(m.Title == "", "title", "must be provided")
	v.Check(len(m.Title) > 500, "title", "must not be more than 500 characters long")

	v.Check(m.Year == 0, "year", "must be provided")
	v.Check(m.Year < 1988, "year", "must be greater than 1988")
	v.Check(m.Year > int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(m.Runtime == 0, "runtime", "must be provided")
	v.Check(m.Runtime < 1, "runtime", "must be a positive integer")

	v.Check(m.Genres == nil, "genres", "must be provided")
	v.Check(len(m.Genres) == 0, "genres", "must contain at least 1 genre")
	v.Check(len(m.Genres) > 5, "genres", "must not be more than 5 genres")
	v.Check(!validator.Unique(m.Genres), "genres", "must not contain duplicate genres")
	v.Check(validator.In("anime", m.Genres), "genres", "we dont accept animes, thank you")

	v.Check(m.Version == 0, "version", "must be provided")
	v.Check(m.Version < 0, "version", "must be a positive integer")
}
