-- name: SessionByToken :one
SELECT id, "lastActivityAt" AS last_activity_at
FROM sessions
WHERE token = @token;

-- name: CreateSession :exec
-- lastActivityAt has no database default, because Prisma set it in the app.
INSERT INTO sessions (token, "lastActivityAt")
VALUES (@token, CURRENT_TIMESTAMP);

-- name: TouchSession :exec
UPDATE sessions
SET "lastActivityAt" = CURRENT_TIMESTAMP
WHERE id = @id;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token = @token;
