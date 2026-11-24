# SQLite notes

Working notes for `sqlite-plan.md`. Facts, measurements, gotchas, and the prototype's findings, so the work can
start without redoing them. Remove notes that stop being true.

## Step 0 (preparing on Postgres)

- Times (done 2026-11-23, `00005_times_from_go.sql`):
  - The migration drops the defaults of `posts."createdAt"`, `"updatedAt"`, `"publishedAt"`,
    `post_html_cache."renderedAt"`, and `sessions."createdAt"`, and truncates every stored time to milliseconds.
    Prisma wrote milliseconds; Go wrote microseconds from cutover until this migration (seen locally:
    `2026-11-22 12:27:59.656983+01`).
  - `CreateDraft`, `CreateSession`, `TouchSession`, and `SavePostHTML` take the time as a parameter. `CreateDraft` no
    longer passes `NULL` for `publishedAt` and `updatedAt` (their defaults used to be the current time), and
    `CreateSession` now sets `createdAt` too.
  - `currentTime()` in `internal/db/db.go` truncates `time.Now()` to milliseconds; the save transaction uses it too.
    `TestStoredTimes` checks the six columns (it reports all six if the truncation goes).
  - Checked on a copy of the production copy through the real server: no defaults or sub-millisecond times left,
    and login, a new draft, and a publishing save store millisecond times.
- Title and slug (done 2026-11-23, `00006_title_slug_not_null.sql`):
  - On the production copy, 442 titles and 516 slugs were `NULL` and none were `''`. The queries mapped `NULL` to
    `''` (`COALESCE` in `Posts`, `Drafts`, `PostForEdit`) and back (`NULLIF` in `UpdatePost`), so Go never saw
    `NULL`.
  - The migration sets them to `''`, then `DEFAULT ''` and `NOT NULL`. The queries read and write the columns
    directly; the seeds insert `''`. sqlc's types didn't change (`string`).
  - Checked on a copy of the production copy through the real server: all 670 titles and slugs equal the old
    `COALESCE` values; `/669` redirects to its slug, `/670` (no slug) loads, and a save that clears the title and
    slug stores `''` and makes the old slug URL redirect to `/669`.
- Tag order (decided 2026-11-23: keep the order typed in the editor):
  - Today nothing defines it. Saves keep the typed order (`parseTags`, then delete and reinsert the links; SvelteKit
    did the same with `createMany`), but the queries read tags without `ORDER BY`. On production, Postgres reads
    them through the `(postId, tagName)` index, so pages show them by name; locally, a sequential scan returns
    roughly the typed order (posts #1 and #3).
  - A `position` column on `posts_tags` keeps the typed order. The migration fills it from the rows' physical order,
    the best record of the typed order left (inserts append; space freed by deletes can break that, rarely):

    ```sql
    ALTER TABLE posts_tags ADD COLUMN "position" INTEGER;
    UPDATE posts_tags pt
    SET "position" = o.position
    FROM (SELECT ctid, row_number() OVER (PARTITION BY "postId" ORDER BY ctid) - 1 AS position FROM posts_tags) o
    WHERE pt.ctid = o.ctid;
    ALTER TABLE posts_tags ALTER COLUMN "position" SET NOT NULL;
    ```

  - Tested on a copy of the production copy (2026-11-23): positions match the physical order for every post, no
    duplicates. 111 of the 363 posts with several tags then show them in another order than by name (like #664
    `github, cloud`, #6 `internet, business, informazione`): a visible change, back to the order typed.
  - Queries: `ORDER BY pt."position"` inside the `ARRAY(…)` subqueries, and `AddPostTags` numbers the names:
    `SELECT @post_id, t.name, (t.ordinality - 1)::int FROM unnest(@names::text[]) WITH ORDINALITY AS t(name,
    ordinality)`. sqlc accepts it, and `Names` stays `[]string`.
  - SQLite: `position INTEGER NOT NULL` in `posts_tags`; `json_each`'s `key` is the array index, so
    `SELECT @post_id, value, key FROM json_each(CAST(@names AS TEXT))` (sqlc accepts it too). The copy command
    copies the column.
  - The primary key stays `(postId, tagName)`: a tag appears once per post, and `parseTags` already drops repeats.
- Archive (done 2026-11-23):
  - `PublishedTimes` replaces the `EXTRACT` query; `db.Archive(ctx, loc)` counts the months in Go (`countByMonth`),
    and the server passes `views.Rome` (exported for this). The month page (`/archivio/{year}/{month}`) selects
    the month from midnight in Rome too, so the sidebar count and the page agree. Both TODOs are gone.
  - No production post changes month: none falls in the hours where UTC and Rome are in different months
    (checked on the production copy). Checked through the real server by moving one post to 22:30 UTC on
    30 September: the sidebar and both month pages put it in October (22 and 102 posts).
  - Tests: `TestCountByMonth` (midnight in winter and summer time), and `TestFilters` checks February, March (ends
    in summer time), and December in Rome.
  - Cost: every page with the sidebar reads one time per published post (670, about one more a day). On the local
    production copy with pgx it's cheaper than the old query: 0.20 ms per call against 0.47 ms (Postgres executes
    it in 0.21 ms against 0.52 ms, since it skips `EXTRACT`, grouping, and sorting), but allocates 28 KB against
    1.7 KB. If it ever matters: cache the sidebar's archive and tags in memory, recomputed when a save, publish,
    unpublish, or delete changes posts (one server instance makes that safe).
  - Grouping by Rome's month in SQL instead (measured 2026-11-23 on the prototype's ncruces copy):

    | Archive on SQLite | Time per call | Allocated |
    |---|---|---|
    | Times to Go, counted in Go (`countByMonth`) | 0.40 ms | 190 KB |
    | `GROUP BY` a Go function, `rome_month(published_at)` | 1.11 ms | 935 KB |
    | `GROUP BY strftime('%Y%m', published_at, 'localtime')` | 0.40–0.45 ms | 1 KB |

    - A Go function: `conn.CreateFunction("rome_month", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS, fn)` in the
      driver's connection callback, next to `unicode.Register`. `fn` reads the argument with
      `arg[0].Time(sqlite3.TimeFormatAuto)`, converts it to Rome, and returns `yyyymm` with `ctx.ResultInt`; it
      must handle NULL. sqlc accepts the unknown function; `CAST(… AS INTEGER)` types the column. The result equals
      `countByMonth`'s, but it's slower: every row still crosses into Go and parses the text time again. The
      function exists only in the server's connections: the `sqlite3` CLI gets "no such function", so it must never
      appear in views, indexes, or generated columns.
    - `'localtime'`: SQLite's own conversion, as fast as Go and allocating almost nothing, but ncruces implements
      `localtime_r` with Go's `time.Local`, the process's time zone (UTC in the container). 22:30 UTC on
      30 September gave `202609` by default and `202610` only with `TZ=Europe/Rome`. Using it means setting `TZ` on
      Railway and in every test, which changes `time.Local` for the whole server.
    - Kept: counting in Go.
- Rename to snake_case (done 2026-11-23, `00007_snake_case_columns.sql`):
  - `ALTER TABLE … RENAME COLUMN` for every camelCase column (`createdAt`, `updatedAt`, `publishedAt`, `hideTitle`,
    `bodyRevision`, `postId`, `tagName`, `lastActivityAt`, `rendererVersion`, `renderedAt`). The queries lost their
    quotes and the aliases that repeated the new names; sqlc's Go types didn't change.
  - Constraint and index names stay camelCase (`posts_tags_postId_fkey`); `posts_number_key`, which `edit.go`
    checks, has no camelCase. Postgres 18 also named the `NOT NULL` constraints after the old columns
    (`posts_createdAt_not_null`). `position` needs no quotes: Postgres reserves it only as a function or type name.
  - The editor's JSON and form fields (`hideTitle`, `createdAt`, … in `internal/server/edit.go`) aren't columns
    and keep their names.
  - Older deployments read the old names: a rollback past this migration breaks, as with any migration.
- Skipped (2026-11-23): a unique index on `sessions.token`. The SQLite baseline has it (`UNIQUE`).

## Steps 1 and 2 (done 2026-11-23)

- Code:
  - `00001_baseline.sql` replaces the seven Postgres migrations (see "Schema").
  - `db.Open(ctx, path)` and `db.Migrate(ctx, path)` share `openSQL`, which sets the driver parameters and
    registers `ext/unicode`; `SetMaxOpenConns(2)`. SQLite creates a missing file but not a missing directory.
  - `db.Post` and `db.EditablePost` are structs now, built from sqlc's rows (`newPost`, `newEditablePost`) after
    decoding the tags' JSON. `Drafts` converts `DraftsRow` to `PostsRow` (same fields) and reuses `newPost`.
  - `SavePost` opens the transaction and calls `savePost` with `db.queries.WithTx(tx)`; `PostForSave` replaces
    `LockPost` (`FOR UPDATE`). The number taken is `errors.Is(err, sqlite3.CONSTRAINT_UNIQUE)`: ncruces' `Error`
    implements `Is` for `ExtendedErrorCode`.
  - Times are `db.Time` in sqlc's code (see "Schema"); the wrappers convert to `time.Time`, so `db.Session` is a
    struct too. `UpdatePostParams.UpdatedAt` is `*Time` (the column is nullable).
- Tests:
  - `TestMain` builds a template (migrate and seed, or `VACUUM INTO` from `TEST_DATABASE_PATH` opened with
    `mode=ro`, then migrate), and `openTestDB` copies it into `t.TempDir()`. `openTestTx`, the `-short` skip, and
    `TestMigrationsLoad` (TestMain migrates anyway) are gone. The Go tests no longer need Docker.
  - `TestStoredTimes` reads the stored text and checks its exact format and that it's the current time in UTC.
    `TestTimeColumns` checks that every table is `STRICT` and every `*_at` column has the format `CHECK` (both
    failures checked), and that the `CHECK` rejects `2026-11-23T10:00:00Z`. `TestSearchIgnoresCase` (seed post 4 has `più`) fails without `unicode.Register`
    (checked).
  - Browser tests: `DatabaseSync(path, { timeout: 5000 })` seeds while the server holds the file; teardown waits
    for the server to exit before removing the temporary directory. `data.ts` writes times as
    `toISOString()` without `T` and `Z`, the stored format.
- `npm` refuses to run on Node 24 (the nvm default here): `engines` wants 26.x
  (`~/.nvm/versions/node/v26.13.1`).
- Smoke test of the built server on a new file: it migrates, serves `/`, the `sqlite3` CLI reads the file while it
  runs (WAL), and a clean shutdown checkpoints and removes `-wal` and `-shm`.
- The local `.env` still has `DATABASE_URL`: `npm run dev` needs `DATABASE_PATH` (`.env.example`; `/*.db*` is
  ignored by git).

## Step 3 (done 2026-11-23)

- Source: no direct connection to production. Production writes without edits only `post_html_cache` (a view of a
  post whose cache is missing or stale, `postItems`), `sessions` (login, logout, activity once a minute), and
  `goose_db_version` (deploys): losing those costs a re-render or a login. So a backup taken after the last edit
  is enough: run the backups repo's workflow by hand (`workflow_dispatch`), fetch `schema.sql` and `data.sql`
  from its commit with `gh api …/contents/<file>?ref=<sha> -H "Accept: application/vnd.github.raw"`, and import
  them into a throwaway database (`createdb`, `psql -v ON_ERROR_STOP=1 -f schema.sql -f data.sql`).
- The 2026-11-23 backup (commit `1732ecf`, after 00004–00007 deployed): 676 posts, all published; 99 tags; 1,109
  links; 11 sessions; 676 cached pages at renderer version 2; every time in milliseconds.
- `cmd/pgcopy <postgres URL> <sqlite file>`: refuses an existing file, migrates it, reads Postgres in one
  read-only repeatable-read transaction, inserts every table in one SQLite transaction (ids kept; times as
  `db.Time`, bools bound as 0/1), then compares row counts, runs `integrity_check` and `foreign_key_check`, and
  checkpoints the WAL. 122 ms. Removed after the cutover (2026-11-24; it's in git history); pgx stays in `go.mod`
  only as an indirect dependency of sqlc, and the server binary doesn't contain it.
- Comparison: the old server built from `origin/main` (`136663c`, what production runs: `git archive` into the
  scratchpad, symlinked `node_modules` and `.libvips`, `vite build`, `go build`) on the imported database, the new
  one on a copy of the file, both with the same variables. A Python script fetched 2,574 paths logged out (every
  post, its slug and a wrong slug, every home page, category, tag, month, `?page=` edge cases, missing pages, the
  feed, and 200 words with accented letters searched upper-cased and without accents, plus `%`, `_`, `\`, `'`) and
  75 logged in (drafts, editor pages, searches), comparing status, `Location`, `Content-Type`, `Cache-Control`,
  and the body byte for byte. All equal.
- The one difference found: the editor's JSON times. pgx returns times in the process's zone (`+01:00` on this
  Mac), `db.Time` in UTC (`Z`): the same instants, and Railway runs in UTC, so production already sends `Z`. With
  the old server under `TZ=UTC`, every page matched.
- `TEST_DATABASE_PATH=<copy> go test ./internal/db` passes, and the file's checksum doesn't change.
- Note: the work is on the local branch `sqlite`; `git archive HEAD` there gives the new code, not production's.

## Cutover (2026-11-24)

- Backup `de4a171` of the backups repo (run by hand at 21:30 UTC; the last edit was 20:05 UTC): 684 posts, all
  published; 99 tags; 1,122 links; 11 sessions; 684 cached pages. `gh` (run through 1Password's `op plugin`) hung
  with no prompt, so the files came from `git clone --depth 1` over SSH instead.
- `pgcopy` into a new file: every count equal, checks passed, 107 ms.
- Upload: the official `litestream/litestream:0.5.18` image with `litestream.yml` minus `exec`, on a Docker volume
  holding the file, `--env-file .env.r2-prod`; the snapshot (1 MB compressed) was up in 2 s, then `docker stop`.
  A `litestream restore` into another file gave an identical `.dump` of all five tables.
- Railway deploy `355bac0d` (commit `15fce32`, root image) with an empty volume at `/data`: "restore completed" in
  1.6 s, then the server (Railway sets `PORT=8080`). Litestream logged "detected database behind replica" and
  fetched the latest L0 file, then replicated normally.
- Production compared with a local server on the same file: 2,596 logged-out paths, 0 differences (6 minutes).
- A save of post 684 reached the bucket: a restore shows its new `updated_at` and body revision.
- Rollback: the Postgres service is unchanged; the previous deployment needs `DATABASE_URL` back, and anything
  written to SQLite since would have to be copied back by hand.

## Prototype (2026-11-22)

- Built in the session's scratchpad (`…/scratchpad/sqlite`, likely gone by now): a separate Go module with the
  schema, every query, a store wrapping sqlc's code, a Postgres → SQLite copy command, and checks against
  `notedimatteo_prod` (the local copy of the 2026-11-20 backup). The snippets below hold what matters.
- It ran with both drivers through build tags (`open_ncruces.go`, `open_modernc.go`), to compare them on the same code.
- sqlc ran as the repo's pinned tool: `go tool sqlc -f <dir>/sqlc.yaml generate`, from the repo.

### Results

- Copy: 670 posts, 99 tags, 1,095 post–tag links, 11 sessions, 670 cached HTML rows in 120 ms. The file is 1.5 MB
  (the Postgres database is 9.8 MB). `PRAGMA foreign_key_check` is clean.
- Compared with Postgres (the repo's generated pgx code against `notedimatteo_prod`):
  - `Categories`, `Tags`, and `Archive` are equal. `Archive` was grouped in Go in Europe/Rome; local Postgres runs
    in Europe/Rome, so the grouping matched.
  - 326 filters: everything, each category, each of the 87 published tags, each of 15 months, numbers
    0/1/152/670/671, special searches (`%`, `_`, `\`, `'`), and 195 upper-cased words with accented letters taken
    from the bodies. Checked: the same post numbers in the same order, fields, times, tags (sorted), cached HTML,
    `CountPosts` equal to the length, and page 2 (`LIMIT 20 OFFSET 20`).
  - ncruces: 0 mismatches. modernc with its built-in `LIKE`: 201 (ASCII-only case folding); with a Go `LIKE`, 0.
- Save checks (on a copy of the file): number taken → `ErrNumberTaken`; draft → save with new tags (`["go","go-2",
  "nuovo-tag-è"]`) → publish → shown first on the home page with cached HTML at revision 3 → unpublish → delete
  (tags and cache cascade); a published post isn't deleted; foreign keys enforced.
- Concurrency: 8 goroutines × 25 saves on one post while 8 goroutines read the home page. 0 errors, revision
  201 (1 + 200: every save counted). ncruces 135 ms with ~1,000 reads; modernc 256 ms with ~600.

### Timings (MacBook, production copy, 4 connections)

| Workload | ncruces | modernc |
|---|---|---|
| Home page: `Posts` 20, `CountPosts`, `Categories`, `Tags`, `Archive` | 1,297 µs | 1,248 µs |
| One post by number | 166 µs | 156 µs |
| Tag page | 362 µs | 352 µs |
| Search `perché`, page and count | 10.5 ms | 10.8 ms |
| Search without matches (full scan) | 8.0 ms | 8.2 ms |
| All 670 posts | 3.8 ms | 3.8 ms |
| 800 home pages, 8 goroutines | 354 ms | 630 ms |

- Postgres for comparison (`EXPLAIN ANALYZE`, server side only): search `perché` 5.8 ms, no match 4.3 ms, home
  page list 0.23 ms.
- First query after process start, in Linux: ncruces 13 ms, modernc 9 ms, pgx 71 ms.

### Memory (Linux container, arm64, stripped binaries)

Workload: 1,200 home pages, 120 searches, and 120 reads of all posts, 4 goroutines. RSS / peak RSS:

| Connections | ncruces | modernc | pgx (client only) |
|---|---|---|---|
| 1 | 18.1 / 20.0 MB | 18.8 / 19.9 MB | |
| 2 | 20.3 / 23.3 MB | 21.9 / 25.9 MB | |
| 4 | 24.3 / 29.3 MB | 31.4 / 39.1 MB | 16.4 / 20.8 MB |

- Each SQLite connection has its own page cache (2 MB by default, `DEFAULT_CACHE_SIZE=-2000`); ncruces also has a
  Wasm memory per connection (its README warns memory is higher). 2 connections is the plan's choice: WAL lets
  them read while the other writes.
- The pgx numbers leave out the Postgres server, a separate Railway service.

### Binary size (Linux arm64, `-s -w`)

- Shared base (`database/sql`, goose, `net/http`): 3.8 MB. modernc: 6.8 MB. ncruces with `ext/unicode` and
  `ext/fts5`: 10.4 MB. pgx: 8.6 MB.

## Drivers

### ncruces/go-sqlite3 (chosen)

- v0.38.0 (2026-11-21), SQLite 3.54.1. Before 1.0: the API can change between minor versions.
- Since v0.33.0 (2026-03) it translates SQLite's Wasm build to Go with wasm2go, in
  `github.com/ncruces/go-sqlite3-wasm/v6`. Before that it ran the Wasm in wazero. Its only dependencies are Go and
  `x/sys` (`ext/unicode` adds `x/text`).
- sqlc depends on v0.32.0, but the tools share the module graph: sqlc builds with v0.38.0 (checked). It imports
  the deprecated `go-sqlite3/embed`, which now prints "If you're reading this, you're unnecessarily importing
  github.com/ncruces/go-sqlite3/embed." on every `sqlc generate`: harmless, until sqlc drops the import.
- `ext/unicode`: Unicode-aware `upper`, `lower`, `LIKE` (`strings.EqualFold` rules, through a regexp cached per
  statement), `REGEXP`, collations, plus `casefold`, `unaccent`, `normalize`, `initcap`. Overriding `LIKE` turns off
  SQLite's LIKE optimization (indexes for prefix patterns): no query here uses it.
- FTS5 isn't compiled in: `ext/fts5.Register(conn)` adds it per connection, and allows tokenizers written in Go
  (an Italian stemmer, if word search ever needs it). `dbstat` isn't available.
- `_txlock` defaults to `immediate`. A busy timeout of 1 minute is set before any `_pragma`.
- `_timefmt` takes SQLite's formats; for formats without an offset (like `2006-01-02 15:04:05.000`,
  `TimeFormat4`) it converts to UTC before writing. Columns declared `DATETIME`, `DATE`, `TIME`, or `TIMESTAMP`
  scan into `time.Time` (`scanFromDecl` in `driver/driver.go`); a `TEXT` column always scans as a string, and only
  the default RFC 3339 format is recognized in other columns (`maybeTime`, not fixed width). The schema is
  `STRICT`, so the code doesn't use `_timefmt`: `db.Time` writes and reads the text.
- Errors: `*sqlite3.Error`, with `ExtendedCode() == sqlite3.CONSTRAINT_UNIQUE` for the number taken.

The opener used by the prototype:

```go
q := url.Values{}
q.Set("_txlock", "immediate")
q.Set("_timefmt", "2006-01-02 15:04:05.000") // fixed width and UTC: compares and sorts as text
q.Add("_pragma", "busy_timeout(5000)")
q.Add("_pragma", "journal_mode(wal)")
q.Add("_pragma", "synchronous(normal)")
q.Add("_pragma", "foreign_keys(on)") // per connection: without it, cascades silently don't happen
db, err := driver.Open("file:"+path+"?"+q.Encode(), func(c *sqlite3.Conn) error {
	return unicode.Register(c)
})
```

### mattn/go-sqlite3 (rejected, but close)

- v1.14.53 (2026-11-05), SQLite 3.54.1 compiled from the C amalgamation with cgo. Called "(still) the de-facto
  standard" by `cvilsmeier/go-sqlite-bench` and "the most widely used" by jacob.gold's best-practices post (2025-08).
- Tested 2026-11-22 with the same prototype: the same results as Postgres on the 326 filters, and every save check
  passes (200 concurrent saves, 0 errors).
- Its `LIKE` folds only ASCII: Unicode needs our own function, registered in a `ConnectHook` with
  `c.RegisterFunc("like", fn, true)`. `fn` must take `...any`: with `string` arguments, a NULL title fails the query
  ("argument must be BLOB or TEXT"). About 30 lines with `like2regex`. `unaccent`, FTS5 (`sqlite_fts5` build tag),
  and ICU (`sqlite_icu`, needs libicu) aren't there by default.
- Times: writes `time.Time` in the value's own zone (`2006-01-02 15:04:05.999999999-07:00`) and has no option to
  write UTC (`_loc` only affects reading). The copy of production stored 324 posts with `+01:00` and 346 with
  `+02:00` (pgx returns local times), and production would write `+00:00`: text comparison and `ORDER BY` go wrong
  for times hours apart in different zones. Every time would need `.UTC()` in our code.
- cgo stays: the first build takes 15–25 s (cached after), Go commands need a C compiler, and phase 4's goal of a
  server without cgo goes away. No pkg-config or external library, unlike libvips.
- Numbers (same setup as above):

  | | mattn | ncruces |
  |---|---|---|
  | Home page (5 queries) | 833 µs | 1,320 µs |
  | One post | 102 µs | 167 µs |
  | Search `perché`, page and count (Go `LIKE` called from C per row) | 12.8 ms | 10.5 ms |
  | 800 home pages, 8 goroutines | 304 ms | 358 ms |
  | RSS / peak, 2 connections (Linux) | 20.8 / 23.4 MB | 21.4 / 23.4 MB |
  | RSS / peak, 4 connections (Linux) | 24.2 / 28.9 MB | 24.4 / 31.3 MB |
  | Linux binary (`-s -w`) | 4.7 MB (links glibc) | 10.4 MB |

- `cvilsmeier/go-sqlite-bench` (2026-03-23, its own schema): "We cannot declare a clear winner, it all depends on
  the use case." mattn is mid-pack in most tests, fastest with large rows, slowest with 8 concurrent readers.
- Switching drivers later stays cheap: everything goes through `database/sql`, so only the opener, the `LIKE`
  function, and the time handling differ (the prototype ran all three drivers on the same queries).

### Settings from jacob.gold's post (2025-08)

- It suggests a 32 MiB page cache, `temp_store = MEMORY`, `PRAGMA optimize` on each connection, 2–8 connections,
  and `STRICT` tables. The database is 1.5 MB, so the default 2 MB cache already holds all of it; the tables are
  `STRICT` (see "Schema"). `PRAGMA optimize` is cheap: consider it.

### modernc.org/sqlite (rejected)

- v1.63.0 (2026-11-22), SQLite 3.54.1, FTS5 and R*Tree built in. Translated from C; the longest track record.
- Its `LIKE` folds only ASCII. A Go `LIKE` needs our code: `sqlite.MustRegisterDeterministicScalarFunction("like",
  -1, fn)`. The registry is keyed by name, so registering arities 2 and 3 separately panics ("a function named like
  is already registered"); -1 covers both. Functions are global, for every connection.
- Times: `_time_format=sqlite` with `_timezone=UTC` writes `2026-11-19 13:30:31.349+00:00`, with trailing zeros
  removed from the fraction. That still sorts as text while every value is UTC (`+` sorts before digits and `.`),
  but it's fragile. There's no fixed-width option; `_time_integer_format=unix_milli` stores integers instead.
- Errors: `*sqlite.Error`, `Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE` (`modernc.org/sqlite/lib`).

## Schema

The schema is `internal/db/migrations/00001_baseline.sql`. The prototype's had declared `DATETIME` and `BOOLEAN`
types and no `CHECK`s; the baseline is `STRICT` (decided 2026-11-23):

- Without `STRICT`, a column stores a value of the wrong type as it is: `'x'` in an `INTEGER` column, `'true'` in a
  boolean. `STRICT` rejects it for every column, including the ones added later, without remembering anything.
- `STRICT` allows only `INT INTEGER REAL TEXT BLOB ANY`. Times are `TEXT` with
  `CHECK (col IS strftime('%Y-%m-%d %H:%M:%f', col))`, which rejects other formats (`T`/`Z`, no milliseconds, an
  offset), integers, and impossible dates like month 13, and accepts NULL (`IS`, not `=`). `TEXT` alone would
  accept any text. Booleans are `INTEGER` with `CHECK (col IN (0, 1))`. Tested with the `sqlite3` CLI and sqlc.
- The driver decodes times only for declared `DATETIME`-like types, so `db.Time` (`internal/db/time.go`) does it:
  `Value` writes `timeFormat` in UTC, `Scan` parses it. `sqlc.yaml` maps each time column to `Time` (`*Time`
  when nullable) and `posts.hide_title` to `bool` by column name. The type has no `import`, because it's in the
  package sqlc writes to: with one, sqlc imports `db` into `db`. Forgetting an override for a new column makes
  sqlc generate `string` (or `int`), which doesn't compile where the code expects a time.
- A new time column needs the `CHECK` (`TestTimeColumns` fails without it, if the name ends in `_at`) and an
  override; a new boolean needs `IN (0, 1)` and an override.
- `STRICT, WITHOUT ROWID` together parse in SQLite and sqlc.
- Why the docs say SQLite has no datetime type: values are stored as NULL, INTEGER, REAL, TEXT, or BLOB, and the
  date functions work on text, Julian day numbers, or Unix time. Outside `STRICT`, a declared type only sets an
  affinity (`DATETIME` gets NUMERIC, hence `CAST('2026-…' AS DATETIME)` = 2026); sqlc and ncruces read the name.
- No time defaults: `CURRENT_TIMESTAMP` writes `2026-11-22 10:00:00`, without milliseconds, which the `CHECK`
  rejects. Queries take the time as a parameter (`CreateSession`, `TouchSession`,
  `CreateDraft`, `SavePostHTML`). If a default is ever needed: `strftime('%Y-%m-%d %H:%M:%f', 'now')`.
- `sessions.token` gained `UNIQUE` (an index for the lookup; Postgres had none).
- `INTEGER PRIMARY KEY` without `AUTOINCREMENT` can reuse the id of a deleted last row (a deleted draft); nothing
  refers to a deleted post's id, since its tags and cache cascade.
- Prisma's times have millisecond precision already, so the format loses nothing.
- Later schema changes: `ADD COLUMN` works; changing a column's type or constraints means rebuilding the table
  (create, copy, drop, rename), and goose's `-- +goose NO TRANSACTION` when foreign keys must be off.
- The goose history restarts at version 1 in the new file; the Postgres migrations go away with Postgres.
- The format is SQLite's own (`datetime()`, `strftime('%Y-%m-%d %H:%M:%f')`), which its docs also call ISO 8601.
  Text in another format compares wrongly with times SQLite computes:
  `'2026-11-23T10:00:00.000Z' < datetime('2026-11-23 12:00:00')` is 0, since `T` sorts after the space. That
  matters for SQL like a session expiry, `last_activity_at < strftime('%Y-%m-%d %H:%M:%f', 'now', '-60 days')`.

## sqlc with the SQLite engine

All 17 queries generate after these workarounds:

1. `ORDER BY` inside an aggregate (`json_group_array(x ORDER BY x)`) doesn't parse, and the error cascades to every
   query in the same file. An ordered subquery works:

   ```sql
   CAST((SELECT json_group_array(tag_name) FROM (
       SELECT pt.tag_name FROM posts_tags pt WHERE pt.post_id = p.id ORDER BY pt.position)) AS TEXT) AS tags
   ```

   (The prototype ordered by `tag_name`; Step 0 adds `position`.)

   Without the `CAST`, the column is `interface{}`; with it, `string`. Go decodes it with `json.Unmarshal`, so
   `db.Post` becomes a struct built from the row instead of an alias of it.
2. Grammar bug: `a OR b LIKE c ESCAPE '\'` fails ("extraneous input 'ESCAPE'"); `a OR (b LIKE c ESCAPE '\')` works.
   Parenthesize every `LIKE … ESCAPE`.
3. sqlc types a parameter from its first use: in `sqlc.narg(x) IS NULL OR col = sqlc.narg(x)` it's
   `interface{}`, in `col = sqlc.narg(x) OR sqlc.narg(x) IS NULL` it's the column's type (`*int`, `*string`,
   `*Time`). The filters compare first, so they need no `CAST`. The prototype used
   `CAST(sqlc.narg(x) AS TEXT) IS NULL`, which can't type times. `json_each(@names)` still needs
   `json_each(CAST(@names AS TEXT))`.
4. Never `CAST(x AS DATETIME)` (or any type name SQLite doesn't know): NUMERIC affinity turns
   `'2026-11-22 10:00:00.000'` into the integer `2026`. Nothing needs it since point 3.
5. Overrides match the type name case-sensitively: list `integer` (expressions, `COUNT`) and `INTEGER` (columns),
   each plain and nullable, to get `int`.
6. `:execrows` still returns `int64`.
7. `LIMIT NULL` isn't "no limit" in SQLite: pass -1.
8. SQLite's `LIKE` has no default escape character: `ESCAPE '\'` is required for `likeEscaper`'s output.
9. `INSERT … SELECT … ON CONFLICT` needs `WHERE true` before `ON CONFLICT` (SQLite's parsing ambiguity):
   `INSERT INTO tags (name) SELECT value FROM json_each(?) WHERE true ON CONFLICT DO NOTHING`.

Query translations:

| Postgres | SQLite |
|---|---|
| `ARRAY(SELECT …)::text[]` | ordered `json_group_array` subquery, decoded in Go |
| `unnest(@names::text[])` | `json_each(CAST(@names AS TEXT))`, names passed as a JSON string |
| `SELECT … FOR UPDATE` | a plain `SELECT` in an immediate transaction (which holds the write lock) |
| `ILIKE` | `LIKE … ESCAPE '\'` with `ext/unicode`'s `LIKE` |
| `EXTRACT(YEAR/MONTH …)` | `SELECT published_at` and grouping in Go |
| `::int`, `::text`, `::timestamptz` | removed (point 3) |
| `pgconn.PgError.ConstraintName` | `sqlite3.Error.ExtendedCode() == CONSTRAINT_UNIQUE` (the message names `posts.number`) |
| `pgx.ErrNoRows`, `pgx.BeginFunc` | `sql.ErrNoRows`, `BeginTx` + `defer Rollback` + `Commit` |

The `Posts` filter, as it generates:

```sql
WHERE p.published_at IS NOT NULL
  AND (p.number = sqlc.narg(number) OR sqlc.narg(number) IS NULL)
  AND (p.category = sqlc.narg(category) OR sqlc.narg(category) IS NULL)
  AND (EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name = sqlc.narg(tag))
          OR sqlc.narg(tag) IS NULL)
  AND ((p.title LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR (p.body LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id
                     AND (pt.tag_name LIKE sqlc.narg(search_pattern) ESCAPE '\'))
          OR sqlc.narg(search_pattern) IS NULL)
  AND (p.published_at >= sqlc.narg(published_from) OR sqlc.narg(published_from) IS NULL)
  AND (p.published_at < sqlc.narg(published_until) OR sqlc.narg(published_until) IS NULL)
ORDER BY p.published_at DESC, p.number DESC
LIMIT @max_rows OFFSET @skip_rows
```

## Search

387 queries on the production copy: every word with an accented letter, upper-cased and also lower-cased without
accents, plus short and common ones. Counts are posts found:

| Query | Unicode `LIKE` (today) | `unaccent` `LIKE` | FTS5 trigram, `remove_diacritics 1` | FTS5 `unicode61`, prefix |
|---|---|---|---|---|
| Time per query | 8.6 ms | 12 ms | 0.5 ms | 0.05 ms |
| `PERCHÉ` | 59 | 60 | 60 | 60 |
| `perche` | 1 | 60 | 60 | 60 |
| `CITTÀ` | 6 | 16 | 16 | 16 |
| `piu` | 9 | 123 | 123 | 122 |
| `go` | 219 | 219 | 219 | 132 |
| `ai` | 415 | 415 | 415 | 220 |
| `e` | 670 | 670 | 670 | 583 |

- Today's semantics (Postgres `ILIKE`): substring, case-insensitive for every letter, accent-sensitive. Unicode
  `LIKE` reproduces them exactly (0 differences on the 326 filters).
- Ignoring accents finds much more with what people type on phones (`piu`, `perche`). `unaccent(text) LIKE
  unaccent(pattern)` is a query change with no schema; FTS5 trigram gives the same results for queries of 3+
  characters, needs triggers to stay in sync and about 3 MB, and falls back to a scan for shorter queries.
- Word search (FTS5 `unicode61`) changes the semantics: `go` no longer finds `algoritmo`. It gives `bm25` ranking
  and `snippet`/`highlight`; FTS5 has no Italian stemmer (Porter is English), but ncruces accepts Go tokenizers.
  A question for phase 5's articles.
- Outside SQLite: Bleve (Italian analyzer, but a heavy dependency and an index to sync), Meilisearch or Typesense
  (another service), an in-memory index in Go (duplicates SQLite). None is worth it at this size.

## Hosting and Litestream

Decided 2026-11-23: Litestream replicates the database to R2, whichever host runs it; the host (Railway or Bunny
Magic Containers) is chosen at Step 4.

### Litestream

- One command runs everything: `litestream replicate -restore-if-db-not-exists -exec server <db path> <replica URL>`
  (flags before the paths, or a config file). Checked in v0.5.18's source (`cmd/litestream/replicate.go`):
  - With `-restore-if-db-not-exists`, it restores from the replica before replicating when the file is missing; when
    the replica has no backup yet, it logs "no backup found, starting fresh" and continues. When the file exists, it
    skips the restore.
  - With `-exec`, it starts the server after the restore and runs it as a subprocess.
- What it gives: continuous backups with point-in-time restore; a lost volume (Bunny's node failure, a mistake)
  costs at most the last seconds of writes; the first deploy restores the copy of production from R2, so the file
  never has to be uploaded to the volume.
- Locks: Litestream (v0.5.18 uses modernc and mattn, so the standard Unix VFS) and the server open the same file.
  ncruces' default build is "compatible with the standard Unix and Windows SQLite VFSes" (`vfs/README.md`): it uses
  OFD locks, which conflict with SQLite's POSIX record locks across processes as `fcntl(2)` specifies. Never build
  with ncruces' `sqlite3_flock` or `sqlite3_dotlk` tags: ncruces warns that incompatible VFSes "will eventually
  corrupt data". The same applies to the `sqlite3` CLI on a live file.
- Set up 2026-11-23 (`litestream.yml`, the Dockerfile's `server` stage), from v0.5.18's source and litestream.io
  (`guides/docker`, `tips`, `reference/config`, fetched with curl and `uvx trafilatura --markdown < page.html`):
  - The image copies the binary from `litestream/litestream:0.5.18` (static, 50 MB: the image goes from about
    146 MB to 201 MB) and runs `litestream replicate`, which reads `/etc/litestream.yml`. Renovate's
    `dockerfile` manager sees the `COPY --from` image; it's in the weekly server-packages rule.
  - The config uses `exec: server`, `restore-if-db-not-exists: true`, and an S3 replica built from variables
    (Litestream expands `${VAR}`, and reads `LITESTREAM_ACCESS_KEY_ID` and `LITESTREAM_SECRET_ACCESS_KEY` by
    itself). The R2 defaults (signed payloads, 2 parallel uploads) apply when the endpoint ends in
    `.r2.cloudflarestorage.com` and uses https, whether it's in the URL or the `endpoint` field. With an endpoint,
    the region defaults to `us-east-1`, which R2 accepts.
  - Retention: snapshots every 24 h, kept 168 h (the default keeps 24 h, so point-in-time restore would only reach
    a day back). Durations take only `s`, `m`, `h`. The backups repo's daily dump keeps the long history.
  - The bucket must be private and only for this: the uploads bucket is public through
    `STORAGE_PUBLIC_BASE_URL`, and the database holds session tokens. Two processes replicating to the same path
    can make it unrestorable (litestream.io/tips): never point a local run at production's bucket.
  - The application side matches litestream.io's tips already: `busy_timeout` 5 s, WAL, `foreign_keys`,
    `synchronous=NORMAL`. `wal_autocheckpoint = 0` is only for tens of thousands of writes per second.
  - On SIGTERM, Litestream signals the server, waits for it to exit, then syncs the last changes (up to 30 s,
    `shutdown-sync-timeout`). A server that exits (a failed migration) stops Litestream too, with an error.
  - litestream.io/guides/docker: the database must be on local storage or a block device, never NFS, SMB, or
    another network filesystem, where locks don't work. Check what Railway's and Bunny's volumes are.
- Tested 2026-11-23 with Docker, rclone's S3 server standing in for R2, and the 2026-11-23 production copy:
  - A volume with the file: "database exists, skipping restore", the server starts, a snapshot (987 KB
    compressed) is uploaded in a second. A login (a new session) is uploaded as transaction 2 within a second.
  - `docker stop`: exits in 0.2 s with code 0, after the server's shutdown and Litestream's.
  - A new empty volume (owned by `nobody`): "restore completed" in 28 ms, then the server starts. The restored file
    passes `integrity_check` and has 676 posts, 1,109 links, 676 cached pages, and 12 sessions (11 plus the login).
  - An empty bucket can't be tested with rclone: it answers 404 to listing a prefix that doesn't exist, which
    Litestream reports as "restore failed: get v0.3.x time bounds". MinIO's images are gone from Docker Hub and
    quay.io. R2 returns an empty list (below).
- Tested 2026-11-23 on R2, with a dev bucket and a token for it only (`.env.r2`, ignored by git and Docker, passed
  with `docker run --env-file`; production gets another bucket and token). rclone reads the same file through
  `RCLONE_CONFIG_R2_*` variables set inside the container (`provider Cloudflare`, `--s3-no-check-bucket`); its
  403 on `GetBucketVersioning` is expected with a token limited to objects.
  - Empty bucket, empty volume: "no backup found, starting fresh" after about 3 s, then the server migrates a new
    file and the first snapshot is uploaded.
  - The cutover rehearsal: a volume with the production copy is uploaded in 2 s; a restore into a new empty volume
    takes 3 s and passes `integrity_check`, with every row.
  - The endpoint was `<account>.eu.r2.cloudflarestorage.com` (an EU bucket): the R2 detection matches its suffix.
  - A write is lost from the bucket when it comes within a second of Litestream starting and the container stops
    in that same second: Litestream opens the database on its first check (`init` in `Sync`, every
    `monitor-interval`, 1 s), and `DB.Close` makes the final sync only once it's open. The write stays in the
    local file and is uploaded at the next start. After that first second, a login followed at once by
    `docker stop` was uploaded by the final sync (checked with debug logs and a restore).
  - Afterwards, the test folder was deleted from the dev bucket (`rclone purge`).
- Backups repo (2026-11-24): its `backup.yml` restores the latest copy with the `litestream/litestream:0.5.18`
  image (`restore -o … "s3://<bucket>/notedimatteo?endpoint=<endpoint>"`), checks `integrity_check` and that
  `posts` has rows, then commits `notedimatteo.sql` (`sqlite3 .dump`) daily at 01:31 UTC and on push. The Postgres
  files are gone (git history keeps them).
  - A token that can only read the production bucket: the keys are repository secrets, the bucket and endpoint
    variables (an account ID gives no access on its own). `.env.r2-prod-ro` has another read-only token for local
    tests.
  - The dump leaves out `sessions` (login tokens; `last_activity_at` changes on every visit), `post_html_cache`
    (rebuilt when missing; half the size), and Litestream's `_litestream_seq` and `_litestream_lock` tables. It
    keeps `goose_db_version`, so a server starts on `sqlite3 new.db < notedimatteo.sql` without migrating. 670 KB,
    one line per row (`.dump` escapes newlines), so an edit is a one-line diff and an unchanged day makes no
    commit.
  - Tested locally: the restore takes 6 s; the dump reloads with `integrity_check` ok, no `foreign_key_check` rows,
    and every post and link. `-timestamp 2026-11-24T21:45:00Z` restores post 684 at revision 10, before the 21:50
    edit (revision 11). A missing file fails the check ("no such table: posts").
  - The runner (Ubuntu 24.04) has `sqlite3` 3.45.1 and Docker. A restored file is in WAL mode: `sqlite3 -readonly`
    can't open it ("unable to open database file").
- Rejected (2026-11-23): the server uploading a daily `VACUUM INTO` snapshot (a day of writes at risk), a one-off
  server mode importing from Postgres when the file is empty, and uploading the file with `railway ssh`.

### Railway

- A service with a volume can't run two deployments at once: the old one stops before the new one starts, so each
  deploy has a short downtime (accepted). One volume per service, no replicas.
- Volumes are mounted as root, when the container starts (not at build or pre-deploy). Railway's documented fix
  for non-root images is `RAILWAY_RUN_UID=0`, which overrides the image's `USER`; for our own image, removing
  `USER nobody` does the same everywhere. Decided 2026-11-24: run as root. As `nobody` the server could still read
  and change the database and read every secret in its environment; running as root mainly makes escaping the
  container easier. The alternative was a startup script that `chown`s the mount and drops to `nobody` with
  `setpriv` (in the image, from util-linux).
- Railway's volume backups (manual or scheduled, incremental, kept 6 days to 89 days) can't be downloaded, and
  restore only into the same project: a second layer at most. The CLI now manages volume files
  (`railway volume files list|download|upload`, `railway volume browse`).
- Shutdown (docs read 2026-11-23, `deployments/deployment-teardown`, `variables/reference`): after SIGTERM, a
  deployment gets `RAILWAY_DEPLOYMENT_DRAINING_SECONDS` before SIGKILL, and the default is **0**. Litestream's
  final sync needs time: set it to 30 (the server takes up to 10 s, then Litestream's final sync up to 30 s).
  `RAILWAY_DEPLOYMENT_OVERLAP_SECONDS` (default 0) doesn't apply with a volume.
- Storage type: not documented (volumes give 3,000 read and 3,000 write IOPS; backups are copy-on-write). Forum
  questions about fsync and locks on volumes have no answer. Railway runs its own Postgres and MySQL on the same
  volumes, which need working locks and fsync, so they aren't a network filesystem in practice.
- Before the move, the private repo `notedimatteo-backups` ran `supabase db dump` daily against Postgres
  (`go-migration-notes.md`, "Production data"); now it dumps the restored SQLite file (see "Litestream").

### Bunny Magic Containers

- Deploys like `../traquantopassa-svelte`: the build workflow pushes the image to GHCR; a deploy workflow runs
  `BunnyWay/actions/container-update-image` (container `app`, the commit's tag) and
  `.github/scripts/wait-for-deployment.js`, which polls `https://api.bunny.net/mc/apps/{id}/overview` until every
  pod runs the new image.
- Persistent volumes (from Bunny's docs and blog, read 2026-11-23):
  - Announced 2026-03-04; a third-party write-up calls them public preview, and Bunny's pages give no status.
  - Bound to a node in a region. If the node is down for maintenance, the pod waits for it; after a hardware failure
    the volume comes back empty. No automatic backups or replication: Bunny recommends exporting to object storage.
    Litestream covers both.
  - Storage: local to the node (a hardware failure "requires a disk replacement", and a pod can't move while its
    volume's node is down), so not a network filesystem.
  - Containers in a pod share a group that owns the volume, and files created in it are group-writable whatever
    the umask (volumes created from August 2026); the top-level directory has the sticky bit. A non-root user
    should be able to write: test with `nobody`.
  - Redeploys (`rolling-updates`, read 2026-11-23): with a volume, the old pod is stopped and its volume unmounted
    before the new pod starts, so a single pod has a short downtime, and two processes never share the file.
    A new pod that doesn't become healthy within ten minutes is removed and the old pods come back.
  - Shutdown (`graceful-shutdown`): SIGTERM to PID 1 (Litestream), then SIGKILL after
    `terminationGracePeriodSeconds`: 30 s for apps created after 2026-02-01, 1 s before (set through the API,
    `PUT /apps/APP_ID/`). A new app gets 30 s.
  - Restarts (`restart-policy`): automatic on a non-zero exit, every few seconds; the container's file system is
    new each time, the volume stays.
  - Sandbox: gVisor. `fcntl` and `flock` are fully supported, and gVisor's `posixLock`
    (`pkg/sentry/syscalls/linux/sys_file.go`) puts OFD locks (ncruces) and process locks (Litestream) in the same
    lock table with different owners, so they conflict as on Linux. Both processes are in the same sandbox. Still
    worth confirming on Bunny, along with the WAL's shared memory (`-shm`, `mmap`).
  - No SSH, upload, or download documented: the restore on start replaces them.
  - 10 MB/s read and write per volume, up to 100 GB, 2 volumes per app, $0.10/GB a month on the provisioned size.
- Deploy (set up 2026-11-23): like tqp, a separate `deploy.yml` triggered by `workflow_run` when Build completes
  on `main`; it runs only if Build succeeded, so every job passed. GitHub runs `workflow_run` workflows only from
  the default branch's copy, so it does nothing until merged. `.github/scripts/deploy.js` replaces both
  `BunnyWay/actions/container-update-image` and tqp's wait script:
  - The action (v0.2.2, `src/action.ts`) makes two calls: `GET /mc/apps/{id}` to find the container's id in
    `containerTemplates`, then `PATCH /mc/apps/{id}/containers/{containerId}` with `{id, imageTag}`. It doesn't
    await the PATCH, so a failed update doesn't fail the step. The script makes the same calls (`{imageTag}`),
    then polls `GET /overview` every 10 s for 15 minutes.
  - Done when the app is `active` and every pod is `ready` with container `app` `ready` on an image ending in
    `:<sha>` (before any `@sha256:` digest). It fails on an app `failing` or `suspended`, or a region `failing`.
    Unlike tqp, `inactive` isn't a failure: with a volume the old pod stops first, so for a while there are no
    pods. It logs each pod's status and messages.
  - Schemas from the OpenAPI spec (`https://api-mc.opsbunny.net/docs/public/swagger.json`, linked from
    `bunny.net/docs/llms.txt`): app status `unknown active progressing inactive failing suspended`, pod
    `notScheduled scheduled ready deleting`, container `notStarted started ready`.
  - Tested with a fake `fetch` (`node --import mock.mjs`): an old pod, no pods (`inactive`), a starting pod, then
    the new one ready → success; a `failing` app → exit 1.
  - The job's `if` uses `vars.BUNNY_APP_ID`, which must be a repository variable: a job's `if` is evaluated
    before its environment's variables load. The API key is a secret of the `production` environment. The tag
    is `github.event.workflow_run.head_sha`, the commit Build tagged the image with.
  - The `docker` job logs in to GHCR with `GITHUB_TOKEN` (`packages: write`) and pushes only on `main`. `gh actions-lock`
    added `deploy.yml`'s checkout to `actions.lock`; `--verify` passes.
- App settings to create (dashboard or API):
  - One container named `app`, image `ghcr.io/matteocontrini/notedimatteo:<tag>`. The GHCR package is private by
    default: connect the registry with a GitHub token that can read packages (Bunny's "Image Registries").
  - One region, exactly one instance, no autoscaling: each pod gets its own volume, and two pods would replicate two
    databases to the same bucket path.
  - A volume mounted at `/data` (1 GB is plenty: the file is 1.5 MB), `DATABASE_PATH=/data/notedimatteo.db`.
  - The variables: `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `STORAGE_PUBLIC_BASE_URL`, the `STORAGE_S3_*` ones,
    `SITE_URL`, and the four `LITESTREAM_*` (the dev bucket while testing, production's after).
  - The endpoint on port 6200, a health check on `/healthz`, and `terminationGracePeriodSeconds` 30 (the default
    for new apps; check it in `GET /apps/{id}`).
- **Litestream fails on Bunny (2026-11-23).** The test app (`wcbidSRYxoXRv7D`, Milan, image `f8c76b7`) started
  fine: "no backup found, starting fresh", migration, server listening, 0 restarts. But every Litestream sync failed
  with `locking protocol (15)` (`SQLITE_PROTOCOL`), with "timeout waiting for db initialization" warnings, so
  nothing reached the bucket. The same image replicates fine on Docker (runc) and against R2.
  - The volume, from `/proc/self/mountinfo` in the pod:
    `/data rw - 9p none rw,trans=fd,...,cache=remote_revalidating,disable_fifo_open,directfs`.
  - With `cache=remote_revalidating` (gVisor's mode for shared gofer mounts, whose files can change outside the
    sandbox), memory-mapped files aren't coherent between processes, so SQLite's WAL index (`-shm`) can't be
    shared. Only `cache=fscache` (exclusive file access) keeps it coherent, since all processes map it through the
    same Sentry. Sources: openclaw PR #172220 (SQLite "database is locked" in gVisor over 9p with a WAL replicator;
    WAL only on `trans=fd` + `cache=fscache`), gke-labs/kube-agents #610 (WAL on a gVisor 9p
    `cache=remote_revalidating` volume corrupted two databases in three days). Older, different cause: gVisor's
    VFS1 lacked `F_GETLK` (gVisor #5113, Litestream #183 on Cloud Run and DigitalOcean App Platform, fixed there in
    2024 with a new runtime).
  - So the driver (ncruces OFD locks vs Litestream's POSIX locks) isn't the cause: modernc or mattn would fail the
    same way. Even the server alone, with 2 connections each mapping `-shm`, isn't proven safe on this mount; a safe
    setup there is one connection without `-shm` (`locking_mode=EXCLUSIVE` or a rollback journal), which rules out
    Litestream.
  - Bunny's logs: they appear in the dashboard minutes late; the API has no endpoint to read them (only syslog
    forwarding, `/log/forwarding`). Mount inspection without a shell: override the command with
    `sh -c "grep ' /data ' /proc/self/mountinfo; exec litestream replicate"`, or a second `busybox` container in
    the pod with the volume mounted.
  - Asked Bunny support (2026-11-23) whether multi-process SQLite on volumes is supported or a setting makes it
    work. Alternatives: Railway (normal containers), or Bunny with in-process snapshots instead of Litestream.
- Fits phase 4 of the redesign, which considers Bunny as the CDN in front of imgproxy.
- Bunny Database ("SQLite-compatible", public preview since 2026-02) would put queries over the network and may not
  run `ext/unicode`: not pursued.
- Rollback after cutover, on either host: the previous deployment reads Postgres, which misses whatever was written
  to SQLite since. Copy it back by hand, or don't write until the new deploy looks right.

## Tests and tooling

- See "Steps 1 and 2" for the tests as built.
- The local `sqlite3` (macOS, 3.54.0) opens the files for ad-hoc queries.
- Renovate had no `postgres:18` rule (the image was only in the test code). ncruces released v0.38.0 on
  2026-11-21; the 3-day minimum age applies to its updates.
