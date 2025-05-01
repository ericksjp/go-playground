package data

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// struct that will hold models of our application
type Models struct {
	Movies interface {
		Insert(movie *Movie) error;
		Get(id int64) (*Movie, error);
		Update(movie *Movie) error;
		Delete(id int64) error;
		List(title string, genres []string, filter Filters) ([]*Movie, Metadada, error)
	}
	Users interface {
		Insert(user *User) error;
		GetByEmail(email string) (*User, error)
		Get(id int64) (*User, error)
		Update(user *User) (error)
		Delete(id int64) error;
		GetForToken(tokenScope string, plainText string) (*User, error)
		GetAllForPermission(permission string) ([]*User, error)
	}
	Token interface {
		Insert(token *Token) error
		DeleteAllForUser(id int64, scope string) error
		New(userID int64, duration time.Duration, scope string) (*Token, error)
	}
	Permissions interface {
		GetAllForUser(userID int64) (Permissions, error)
	}
}

// return a Models struct containing the initialized models
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
		Users:  UserModel{DB: db},
		Token:  TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
	}
}
