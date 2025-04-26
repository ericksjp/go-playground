package data

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ericksjp703/greenlight/internal/validator"
	"github.com/lib/pq"
)

const movieInputSize = 4

type Movie struct {
	ID        *int64     `json:"id"`
	CreatedAt *time.Time `json:"-"`
	Title     *string    `json:"title"`
	Year      *int32     `json:"year,omitempty"`
	Runtime   *Runtime   `json:"runtime,omitempty"`
	Genres    []string   `json:"genres,omitempty"`
	Version   *int32     `json:"version"`
}

// -------------------------------------- DB - CRUD

// struct that wraps a sql.DB connection pool
type MovieModel struct {
	DB *sql.DB
}

func (m MovieModel) Insert(movie *Movie) error {
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, release
	`

	// pq.Array adapts a string[] to pq.StringArray
	args := []any{*movie.Title, *movie.Year, *movie.Runtime, pq.Array(movie.Genres)}

	// executes the query string using the args and assign the return values to
	// some movie camps
	return m.DB.QueryRow(query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	var movie Movie
	query := `
		SELECT id, title, year, runtime, genres, release, created_at FROM movies
		WHERE id = $1
	`

	err := m.DB.QueryRow(query, id).Scan(
		&movie.ID,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
		&movie.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &movie, nil
}

func (m MovieModel) Update(movie *Movie) error {
	query, args := setQueryArgsWithDefinedValues(*movie)
	query = fmt.Sprintf(`UPDATE movies SET "release" = "release" + 1, %s WHERE id = $%d RETURNING title, year, runtime, genres, release; `, query, len(args) + 1)

	args = append(args, *movie.ID)

	err := m.DB.QueryRow(query, args...).Scan(
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrRecordNotFound
		}
		return err
	}

	return nil
}

func (m MovieModel) Delete(id int64) error {
	query := `DELETE FROM movies WHERE id = $1;`

	// the Exec() method return a sql.Result object
	result, err := m.DB.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// ------------------------------------------------------------- sql - helpers

func setQueryArgsWithDefinedValues(input Movie) (string, []any) {
	fields := make(map[string]any)
		
	if input.Title != nil {
		fields["title"] = *input.Title
	}
	if input.Year != nil {
		fields["year"] = *input.Year
	}
	if input.Runtime != nil {
		fields["runtime"] = *input.Runtime
	}
	if input.Genres != nil {
		fields["genres"] = pq.Array(input.Genres)
	}

	setClauses := make([]string, 0, movieInputSize + 1)
	args := make([]any, 0, movieInputSize + 1)
	pos := 1

	for field, value := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, pos))
		args = append(args, value)
		pos++
	}

	query := strings.Join(setClauses, ",")
	return query, args
}

// ------------------------------------------------------------- validation

func (mi *Movie) Validate(v *validator.Validator, optional  ...string)  {
	// check for not wanted 'inputs'
	v.Check(mi.ID != nil, "id", "must not be provided")
	v.Check(mi.Version != nil, "version", "must not be provided")
	v.Check(mi.CreatedAt != nil, "created_at", "must not be provided")

	// check for optional inputs
	if !validator.In("Title", optional) {
		v.Check(mi.Title == nil, "title", "must be provided")
	}
	if !validator.In("Year", optional) {
		v.Check(mi.Year == nil, "year", "must be provided")
	}
	if !validator.In("Runtime", optional) {
		v.Check(mi.Runtime == nil, "runtime", "must be provided")
	}
	if !validator.In("Genres", optional) {
		v.Check(mi.Genres == nil, "genres", "must be provided")
	}

	// regular validation
	mi.validateTitle(v)
	mi.validateYear(v)
	mi.validateRuntime(v)
	mi.validateGenres(v)
}

// ------------------------------------------------------------- validation - specific

func (mi *Movie) validateTitle(v *validator.Validator) {
	if mi.Title == nil {
		return
	}

    v.Check(*mi.Title == "", "title", "must not be empty")
    v.Check(len(*mi.Title) > 500, "title", "must not be more than 500 characters long")
}

func (mi *Movie) validateYear(v *validator.Validator) {
	if mi.Year == nil {
		return
	}

    v.Check(*mi.Year < 1988, "year", "must be greater than 1988")
    v.Check(*mi.Year > int32(time.Now().Year()), "year", "must not be in the future")
}

func (mi *Movie) validateRuntime(v *validator.Validator) {
	if mi.Runtime == nil {
		return
	}

    v.Check(*mi.Runtime < 1, "runtime", "must be a positive integer")
}

func (mi *Movie) validateGenres(v *validator.Validator) {
	if mi.Genres == nil {
		return
	}
    genres := mi.Genres
    
    v.Check(len(genres) == 0, "genres", "must contain at least 1 genre")
    v.Check(len(genres) > 5, "genres", "must not be more than 5 genres")
    v.Check(!validator.Unique(genres), "genres", "must not contain duplicate genres")
    v.Check(validator.In("anime", genres), "genres", "we don't accept animes, thank you")
}
