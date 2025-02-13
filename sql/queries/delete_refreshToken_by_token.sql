-- name: DeleteRefreshTokenByToken :exec
DELETE FROM refresh_tokens
WHERE token = $1;
