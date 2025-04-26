package data

import (
	"database/sql"
	"errors"
)

var ErrRecordNotFound = errors.New("record not found")

// struct that will hold models of our application
type Models struct {
	Movies interface {
		Insert(movie *Movie) error;
		Get(id int64) (*Movie, error);
		Update(movie *Movie) error;
		Delete(id int64) error;
	}
}

// return a Models struct containing the initialized models
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{
			DB: db,
		},
		// Movies: MovieMockModel{},
	}
}
