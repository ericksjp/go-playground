package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/ericksjp703/greenlight/internal/validator"
	"github.com/lib/pq"
)

var ErrRecordNotFound = errors.New("record not found")

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

func (m Movie) Validate(v *validator.Validator) {

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

	v.Check(m.Version != 0, "version", "must be empty")
	// v.Check(m.Version < 0, "version", "must be a positive integer")

	v.Check(m.ID != 0, "id", "must be empty")
}

// struct that wraps a sql.DB connection pool
type MovieModel struct {
	DB *sql.DB
}

// mutates the id and created_at camps
func (m MovieModel) Insert(movie *Movie) error {
	query := `
		INSERT INTO movies (title, year, runtime, genres, release)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	// pq.Array adapts a string[] to pq.StringArray
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres), movie.Version}

	// executes the query string using the args and assign the return values to
	// some movie camps
	return m.DB.QueryRow(query, args...).Scan(&movie.ID, &movie.CreatedAt)
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

// func (m MovieModel) GetAll() ([]Movie, error) {
// 	query := `SELECT "id", "title", "year", "runtime", "genres", "release", "created_at" FROM movies`
//
// 	rows, err := m.DB.Query(query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
//
// 	var movies []Movie
//
// 	for rows.Next() {
// 		var movie Movie
// 		err := rows.Scan(
// 			&movie.ID,
// 			&movie.Title,
// 			&movie.Year,
// 			&movie.Runtime,
// 			pq.Array(&movie.Genres),
// 			&movie.Version,
// 			&movie.CreatedAt,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		movies = append(movies, movie)
// 	}
//
// 	return movies, nil
// }

func (m MovieModel) Update(movie *Movie) error {
	query := `
		UPDATE movies
		SET "release" = "release" + 1, 
			title = $1,
			year = $2,
			runtime = $3,
			genres = $4
		WHERE id = $5
		RETURNING release;
	`

	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres), movie.ID}

	err := m.DB.QueryRow(query, args...).Scan(
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

// type MockMovieModel struct{}
