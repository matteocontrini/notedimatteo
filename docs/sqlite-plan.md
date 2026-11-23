# SQLite plan

Replace PostgreSQL with SQLite, in a file on the Railway service's volume. This runs after `go-migration-plan.md`
and before the redesign's phases 2 (revisions) and 5 (articles), which add tables and columns.

Working notes, measurements, and the prototype's findings: `sqlite-notes.md`.

## Why

- One service fewer on Railway: no separate Postgres to run, pay for, and update.
- Development, Go tests, and production copies work with a file: no Postgres, testcontainers, or `pg_dump`.
- The data is small (670 posts: a 1.5 MB file) and has one writer.

## Decisions

| Area        | Choice                                                                                                              |
|-------------|---------------------------------------------------------------------------------------------------------------------|
| Driver      | `ncruces/go-sqlite3` through `database/sql`, without cgo; `ext/unicode` registered on each connection               |
| Connections | WAL, `synchronous(normal)`, `foreign_keys(on)`, a busy timeout, immediate transactions; 2 connections                |
| sqlc        | `engine: sqlite`, `database/sql`; the workarounds are in the notes                                                  |
| Schema      | A new goose baseline with snake_case columns; declared `DATETIME` and `BOOLEAN` types, so not `STRICT` tables       |
| Times       | UTC text `2006-01-02 15:04:05.000` (the driver's `_timefmt`); Go sets every time, and no column has a time default   |
| Tags        | Kept in the order typed (a `position` column); `json_group_array` decoded in Go                                     |
| Archive     | Months grouped in Go in Europe/Rome (done in Step 0)                                                                |
| Search      | Unicode-aware `LIKE`, so results stay the same as Postgres's `ILIKE`                                                |
| Deploys     | A volume means a short downtime on every deploy: accepted                                                           |

## Steps

### Step 0: Prepare on Postgres

Changes that don't depend on SQLite, checked by today's tests and by production, so the move only changes the
dialect. The revisions work (branch `post-revisions`) merges after the move.

- [x] Go sets every time, rounded to milliseconds; drop the time defaults and `CURRENT_TIMESTAMP` from the queries
  (`00005`)
- [x] Make `title` and `slug` `NOT NULL DEFAULT ''`, and drop `COALESCE` and `NULLIF` (`00006`)
- [x] Keep tags in the order typed: a `position` column on `posts_tags`, filled from the rows' current order (`00004`)
- [x] Group the archive by month in Go, in Europe/Rome; the month pages use Rome too
- [x] Rename the columns to snake_case (`00007`)

### Step 1: Database package

- [ ] Replace the migrations with a SQLite baseline; goose with `DialectSQLite3`, still at server start
- [ ] Open the database with the driver settings above; `DATABASE_PATH` replaces `DATABASE_URL`
- [ ] Port the queries and `sqlc.yaml`
- [ ] Port the wrappers: `sql.ErrNoRows`, tags, archive, the save transaction, the unique-number error
- [ ] Remove pgx and the Postgres testcontainers modules

### Step 2: Tests

- [ ] Go database tests on a temporary file with the seed; a variable points them at a production copy instead
- [ ] Browser tests: the server on a temporary file, seeded with Node's `node:sqlite`; Docker stays for rclone

### Step 3: Copy

- [ ] Add a command that copies a Postgres database into a new SQLite file, sessions included
- [ ] Compare the copy of production with Postgres, like the prototype: lists, filters, searches, pages
- [ ] Decide how the file reaches the volume

### Step 4: Deployment

- [ ] Add the volume, and make it writable: Railway mounts it as root, and the server runs as `nobody`
- [ ] Choose the backups (a snapshot uploaded to R2 by the server, or Litestream), replace the `supabase db dump`
  Action, and test a restore
- [ ] Update `ARCHITECTURE.md`, `README.md`, `.env.example`, CI, and Renovate

### Step 5: Cutover

- [ ] Stop writing, copy production, deploy, and check pages, login, a save, and an upload
- [ ] Keep the Postgres service for a while as the rollback, then delete it
- [ ] Remove the copy command

### Afterwards

- [ ] Optional: make search ignore accents (`piu` finds 9 posts today, 123 with `più`)
