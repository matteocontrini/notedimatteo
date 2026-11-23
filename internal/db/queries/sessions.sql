-- name: SessionByToken :one
SELECT id, last_activity_at
FROM sessions
WHERE token = @token;

-- name: CreateSession :exec
INSERT INTO sessions (token, created_at, last_activity_at)
VALUES (@token, @now::timestamptz, @now::timestamptz);

-- name: TouchSession :exec
UPDATE sessions
SET last_activity_at = @now::timestamptz
WHERE id = @id;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token = @token;
