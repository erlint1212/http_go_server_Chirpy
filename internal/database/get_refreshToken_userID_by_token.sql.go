// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: get_refreshToken_userID_by_token.sql

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const getRefreshTokenUserIDByToken = `-- name: GetRefreshTokenUserIDByToken :one
SELECT user_id, expires_at, revoked_at FROM refresh_tokens
WHERE token = $1
`

type GetRefreshTokenUserIDByTokenRow struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt sql.NullTime
}

func (q *Queries) GetRefreshTokenUserIDByToken(ctx context.Context, token string) (GetRefreshTokenUserIDByTokenRow, error) {
	row := q.db.QueryRowContext(ctx, getRefreshTokenUserIDByToken, token)
	var i GetRefreshTokenUserIDByTokenRow
	err := row.Scan(&i.UserID, &i.ExpiresAt, &i.RevokedAt)
	return i, err
}
