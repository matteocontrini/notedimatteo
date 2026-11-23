# SQLite plan

Replace PostgreSQL with SQLite, in a file on a volume, with Litestream streaming every change to R2. This runs
after `go-migration-plan.md` and before the redesign's phases 2 (revisions) and 5 (articles), which add tables and
columns.

Working notes, measurements, and the prototype's findings: `sqlite-notes.md`.

## Why

- One service fewer: no separate Postgres to run, pay for, and update.
- Development, Go tests, and production copies work with a file: no Postgres, testcontainers, or `pg_dump`.
- The data is small (670 posts: a 1.5 MB file) and has one writer.

## Decisions

| Area        | Choice                                                                                                              |
|-------------|---------------------------------------------------------------------------------------------------------------------|
| Driver      | `ncruces/go-sqlite3` through `database/sql`, without cgo; `ext/unicode` registered on each connection               |
| Connections | WAL, `synchronous(normal)`, `foreign_keys(on)`, a busy timeout, immediate transactions; 2 connections                |
| sqlc        | `engine: sqlite`, `database/sql`; the workarounds are in the notes                                                  |
| Schema      | A new goose baseline with snake_case columns and `STRICT` tables; booleans are `INTEGER` with a `CHECK` |
| Times       | UTC text `2006-01-02 15:04:05.000`, with a `CHECK` on the format; `db.Time` in Go; Go sets every time, and no column has a time default |
| Tags        | Kept in the order typed (a `position` column); `json_group_array` decoded in Go                                     |
| Archive     | Months grouped in Go in Europe/Rome (done in Step 0)                                                                |
| Search      | Unicode-aware `LIKE`, so results stay the same as Postgres's `ILIKE`                                                |
| Hosting     | Railway or Bunny Magic Containers (decided at Step 4); one instance with the file on a volume; a short downtime on every deploy is accepted |
| Litestream  | `litestream replicate -restore-if-db-not-exists -exec server`: restores from R2 when the volume has no database, then streams changes to R2 while running the server |

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

From Step 1 on, the code expects SQLite: production can't deploy it until the cutover, so the work stays off
`main` until Step 5.

### Step 1: Database package

- [x] Replace the migrations with a SQLite baseline; goose with `DialectSQLite3`, still at server start
- [x] Open the database with the driver settings above; `DATABASE_PATH` replaces `DATABASE_URL`
- [x] Port the queries and `sqlc.yaml`
- [x] Port the wrappers: `sql.ErrNoRows`, tags, archive, the save transaction, the unique-number error
- [x] Remove pgx and the Postgres testcontainers modules (Step 3's copy command brings pgx back until Step 5)

### Step 2: Tests

- [x] Go database tests: each test copies a template file with the seed; `TEST_DATABASE_PATH` replaces the seed
  with a copy of another file
- [x] Browser tests: the server on a temporary file, seeded with Node's `node:sqlite`; Docker stays for rclone

### Step 3: Copy

- [x] Add a command that copies a Postgres database into a new SQLite file, sessions included (`cmd/pgcopy`)
- [x] Compare the copy of production with Postgres: every page from the old and new servers is identical

### Step 4: Deployment

- [ ] Add Litestream to the image, running the server with `litestream replicate -restore-if-db-not-exists -exec`,
  and replicating to R2
- [ ] Choose the host: Railway, or Bunny Magic Containers (deployed like `../traquantopassa-svelte`: GHCR, then
  `container-update-image`). For Bunny, try a throwaway app first: can `nobody` write to a fresh volume, what does a
  redeploy do, and does the restore on start work
- [ ] Make the volume writable for `nobody` (Railway mounts it as root)
- [ ] Replace the `supabase db dump` Action: restore from R2 and commit a text dump; test a restore, including to an
  earlier point in time
- [ ] Update `ARCHITECTURE.md`, `README.md`, `.env.example`, CI, and Renovate

### Step 5: Cutover

- [ ] Stop writing, run the backup workflow, import its dump into a local Postgres, copy that with `pgcopy`, and
  replicate the file to R2 with Litestream
- [ ] Deploy with an empty volume, so the first start restores from R2; check pages, login, a save, and an upload
- [ ] Keep the Postgres service for a while as the rollback, then delete it
- [ ] Remove the copy command

### Afterwards

- [ ] Optional: make search ignore accents (`piu` finds 9 posts today, 123 with `più`)
