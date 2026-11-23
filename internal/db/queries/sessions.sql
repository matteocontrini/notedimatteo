-- name: SessionByToken :one
SELECT id, "lastActivityAt" AS last_activity_at
FROM sessions
WHERE token = @token;

-- name: CreateSession :exec
INSERT INTO sessions (token, "createdAt", "lastActivityAt")
VALUES (@token, @now::timestamptz, @now::timestamptz);

-- name: TouchSession :exec
UPDATE sessions
SET "lastActivityAt" = @now::timestamptz
WHERE id = @id;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token = @token;
