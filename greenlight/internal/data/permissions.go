package data

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"github.com/lib/pq"
)

type Permissions []string

func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

type PermissionModel struct {
	DB *sql.DB
}

func (p PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT p.code
		FROM permissions p
		INNER JOIN users_permissions up
		ON p.id = up.permission_id
		WHERE up.user_id = $1`

	var permissions Permissions

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	rows, err := p.DB.QueryContext(ctx, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return permissions, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var permission string
		err = rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func (p PermissionModel) AddForUser(userId int64, permissions ...string) error {
	query := `
		INSERT into users_permissions
		SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
	`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	args := []any{
		userId,
		pq.Array(permissions),
	}

	_, err := p.DB.ExecContext(ctx, query, args...)
	return err
}
