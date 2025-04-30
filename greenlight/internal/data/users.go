package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/ericksjp703/greenlight/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

// Define a custom ErrDuplicateEmail error.
var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

type password struct {
	plaintext *string
	hash      []byte
}

// calculate the bcrypt hash of a plaintext password and stores both the
// plaintext and the hash in the struct
func (p *password) Set(plaintext string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintext
	p.hash = hash

	return nil
}

// checks if the provided plainxted password matches the hashed password stored
// in the struct
func (p *password) Matches(plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintext))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

type User struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email == "", "email", "must be provided")
	v.Check(!validator.Matches(email, validator.EmailRX), "email", "must be a valid email adress")
}

func ValidatePassword(v *validator.Validator, password string) {
	v.Check(password == "", "password", "must be provided")
	v.Check(len(password) < 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) > 72, "password", "must be at max 72 bytes long")
}

func ValidateTokenPlaintext(v *validator.Validator, plaintextToken string) {
	v.Check(plaintextToken == "", "token", "must not be empty")
	v.Check(len(plaintextToken) != 26, "token", "invalid or expired activation token")
}

type UserModel struct {
	DB *sql.DB
}

func (m UserModel) Insert(user *User) error {
	query := `
		INSERT into users (name, email, password_hash, activated)
		values ($1, $2, $3, $4)
		RETURNING id, created_at, version
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
	}
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) getUser(query string, args ...any) (*User, error) {
	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) Get(id int64) (*User, error) {
	query := `
		SELECT id, name, email, password_hash, activated, version
		FROM users
		WHERE id = $1
	`
	return m.getUser(query, id)
}

func (m UserModel) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, name, email, password_hash, activated, version
		FROM users
		WHERE email = $1
	`
	return m.getUser(query, email)
}

func (m UserModel) Update(user *User) error {
	// optimistic concurrency control using the version column
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version
	`

	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (u UserModel) Delete(id int64) error {
	query := `
		DELETE FROM users
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := u.DB.ExecContext(ctx, query, id)
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

func (u UserModel) GetForToken(tokenScope string, tokenPlainText string) (*User, error) {
	// calculate the sha-256 hash for the plaintext token
	tokenHash := sha256.Sum256([]byte(tokenPlainText))

	query := `
		SELECT u.id, u.name, u.email, u.password_hash, u.activated, u.version
		FROM users u
		INNER JOIN tokens t
		ON u.id = t.user_id
		WHERE t.hash = $1
		AND t.scope = $2
		AND t.expiry > $3
	`

	// transform the tokenHash arr into a slice
	// pass the current time to check againt the expiry time of the token
	args := []any{tokenHash[:], tokenScope, time.Now()}

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := u.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

// ------------------------------------------------------------- User Input

type UserInput struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

// updates the fields of a User struct based on the non-nil fields of a UserInput struct.
func (ui *UserInput) UpdateUserFields(user *User) error {
	if ui.Name != nil {
		user.Name = *ui.Name
	}
	if ui.Email != nil {
		user.Email = *ui.Email
	}
	if ui.Password != nil {
		if err := user.Password.Set(*ui.Password); err != nil {
			return err
		}
	}
	return nil
}

func (ui *UserInput) Validate(v *validator.Validator, optional ...string) {

	if ui.Name == nil {
		v.Check(!validator.In("Name", optional), "name", "must be provided")
	} else {
		ui.validateName(v)
	}

	if ui.Email == nil {
		v.Check(!validator.In("Email", optional), "email", "must be provided")
	} else {
		ValidateEmail(v, *ui.Email)
	}

	if ui.Password == nil {
		v.Check(!validator.In("Password", optional), "password", "must be provided")
	} else {
		ValidatePassword(v, *ui.Password)
	}
}

func (ui *UserInput) validateName(v *validator.Validator) {
	v.Check(*ui.Name == "", "name", "must not be empty")
	v.Check(len(*ui.Name) > 500, "name", "must not be more than 500 characters long")
}
