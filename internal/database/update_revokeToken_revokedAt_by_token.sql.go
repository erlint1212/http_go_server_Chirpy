// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: update_revokeToken_revokedAt_by_token.sql

package database

import (
	"context"
	"database/sql"
)

const updateRefreshTokenRevokedAtByToken = `-- name: UpdateRefreshTokenRevokedAtByToken :exec
UPDATE refresh_tokens
SET revoked_at = $2, updated_at = $2
WHERE token = $1
`

type UpdateRefreshTokenRevokedAtByTokenParams struct {
	Token     string
	RevokedAt sql.NullTime
}

func (q *Queries) UpdateRefreshTokenRevokedAtByToken(ctx context.Context, arg UpdateRefreshTokenRevokedAtByTokenParams) error {
	_, err := q.db.ExecContext(ctx, updateRefreshTokenRevokedAtByToken, arg.Token, arg.RevokedAt)
	return err
}
