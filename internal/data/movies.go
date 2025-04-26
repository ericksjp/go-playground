package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ericksjp703/greenlight/internal/validator"
	"github.com/lib/pq"
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
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// executes the query string using the args and assign the return values to
	// some movie camps
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	var movie Movie
	query := `
		SELECT pg_sleep(10), id, title, year, runtime, genres, release, created_at FROM movies
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&[]byte{},
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
	// optimistic concurrency control using the version column
	query := `
		UPDATE movies
		SET title = $1, year = $2, runtime = $3, genres = $4, release = release + 1
		WHERE id = $5 AND release = $6
		RETURNING release
	`

	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}

	return nil
}

func (m MovieModel) Delete(id int64) error {
	query := `DELETE FROM movies WHERE id = $1;`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// the Exec() method return a sql.Result object
	result, err := m.DB.ExecContext(ctx, query, id)
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

// ------------------------------------------------------------- Movie Input

type MovieInput struct {
	Title     *string    `json:"title"`
	Year      *int32     `json:"year,omitempty"`
	Runtime   *Runtime   `json:"runtime,omitempty"`
	Genres    []string   `json:"genres,omitempty"`
}

func (mi *MovieInput) ApplyUpdates(movie *Movie) {
	if mi.Title != nil {
		movie.Title = *mi.Title
	}
	if mi.Year != nil {
		movie.Year = *mi.Year
	}
	if mi.Runtime != nil {
		movie.Runtime = *mi.Runtime
	}
	if mi.Genres != nil {
		movie.Genres = mi.Genres
	}
}

func (mi *MovieInput) Validate(v *validator.Validator, optional ...string) {
	
	if mi.Title == nil {
		v.Check(!validator.In("Title", optional), "title", "must be provided")
	} else {
		mi.validateTitle(v)
	}

	if mi.Year == nil {
		v.Check(!validator.In("Year", optional) , "year", "must be provided")
	} else {
		mi.validateYear(v)
	}

	if mi.Runtime == nil {
		v.Check(!validator.In("Runtime", optional) , "runtime", "must be provided")
	} else {
		mi.validateRuntime(v)
	}

	if mi.Genres == nil {
		v.Check(!validator.In("Genres", optional) , "genres", "must be provided")
	} else {
		mi.validateGenres(v)
	}
}

// ------------------------------------------------------------- validation - specific

func (mi *MovieInput) validateTitle(v *validator.Validator) {
	v.Check(*mi.Title == "", "title", "must not be empty")
    v.Check(len(*mi.Title) > 500, "title", "must not be more than 500 characters long")
}

func (mi *MovieInput) validateYear(v *validator.Validator) {
	v.Check(*mi.Year < 1988, "year", "must be greater than 1988")
    v.Check(*mi.Year > int32(time.Now().Year()), "year", "must not be in the future")
}

func (mi *MovieInput) validateRuntime(v *validator.Validator) {
	v.Check(*mi.Runtime < 1, "runtime", "must be a positive integer")
}

func (mi *MovieInput) validateGenres(v *validator.Validator) {
	v.Check(len(mi.Genres) == 0, "genres", "must contain at least 1 genre")
    v.Check(len(mi.Genres) > 5, "genres", "must not be more than 5 genres")
    v.Check(!validator.Unique(mi.Genres), "genres", "must not contain duplicate genres")
    v.Check(validator.In("anime", mi.Genres), "genres", "we don't accept animes, thank you")
}
