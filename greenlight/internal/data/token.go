package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"
)

const (
	ScopeActivation = "activation"
)

type Token struct {
	UserID    int64
	Plaintext string
	Hash      []byte
	Expiry    time.Time
	Scope     string
}

// generate a new token for a user with a specific duration and scope 
// this token is 26 bytes long
func generateToken(userId int64, duration time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userId,
		Expiry: time.Now().Add(duration),
		Scope:  scope,
	}

	// define the entropy point for the random bytes
	randomBytes := make([]byte, 16)

	// fill bytes slice with random bytes
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// encode the random bytes to base32 string without padding
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// create a sha256 hash of the plaintext token
	hash := sha256.Sum256([]byte(token.Plaintext))

	// convert the hash to a byte slice and assign it to the token
	token.Hash = hash[:]

	return token, nil
}

type TokenModel struct {
	DB *sql.DB
}

// shortcut to create a new token and insert it into the database
func (m TokenModel) New(userID int64, duration time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, duration, scope)
	if err != nil {
		return nil, err
	}
	err = m.Insert(token)
	return token, err
}

// Insert() adds the data for a specific token to the tokens table.
func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)`

	args := []any{
		token.Hash,
		token.UserID,
		token.Expiry,
		token.Scope,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

// delete all tokens of a specific user for a specific scope
func (m TokenModel) DeleteAllForUser(id int64, scope string) error {
	query := `
		DELETE FROM tokens
		WHERE user_id = $1 AND scope = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id, scope)
	return err
}
