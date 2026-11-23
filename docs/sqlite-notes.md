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
- R2 needs its endpoint in the replica URL or a config file: check the v0.5 syntax.
- Locks: Litestream (v0.5.18 uses modernc and mattn, so the standard Unix VFS) and the server open the same file.
  ncruces' default build is "compatible with the standard Unix and Windows SQLite VFSes" (`vfs/README.md`): it uses
  OFD locks, which conflict with SQLite's POSIX record locks across processes as `fcntl(2)` specifies. Never build
  with ncruces' `sqlite3_flock` or `sqlite3_dotlk` tags: ncruces warns that incompatible VFSes "will eventually
  corrupt data". The same applies to the `sqlite3` CLI on a live file.
- To check while setting it up: Litestream's guidance for applications (checkpointing, busy timeout), retention
  settings, and how its binary gets into the image (release download or `go install`), so Renovate can track it.
- Backups repo: its Action can run `litestream restore` from R2 into a file and commit `sqlite3 .dump` output,
  keeping today's text history. It needs a read-only R2 token.
- Rejected (2026-11-23): the server uploading a daily `VACUUM INTO` snapshot (a day of writes at risk), a one-off
  server mode importing from Postgres when the file is empty, and uploading the file with `railway ssh`.

### Railway

- A service with a volume can't run two deployments at once: the old one stops before the new one starts, so each
  deploy has a short downtime (accepted). One volume per service, no replicas.
- Volumes are mounted as root, when the container starts (not at build or pre-deploy). The image runs as `nobody`.
  Railway's only documented fix is `RAILWAY_RUN_UID=0`, which runs the server as root. To keep `nobody`: start as
  root, `chown` the mount, and drop to `nobody` (Debian's `setpriv`, in util-linux).
- Railway's volume backups (manual or scheduled, incremental) can't be downloaded: a second layer at most.
- Today's backups: the private repo `notedimatteo-backups` runs `supabase db dump` daily against Postgres
  (`go-migration-notes.md`, "Production data").

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
  - Containers in a pod share a group that owns the volume, and new files are group-writable; fresh volumes'
    owner and mode aren't documented. Test whether `nobody` can write.
  - Redeploy behavior (stop before start, or overlap) isn't documented. Two processes writing the same file through
    different volumes would break replication: test it.
  - No SSH, upload, or download documented: the restore on start replaces them.
  - 10 MB/s read and write per volume, up to 100 GB, 2 volumes per app, $0.10/GB a month on the provisioned size.
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
