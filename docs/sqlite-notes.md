# SQLite notes

Working notes for `sqlite-plan.md`. Facts, measurements, gotchas, and the prototype's findings, so the work can
start without redoing them. Remove notes that stop being true.

## Step 0 (preparing on Postgres)

- Times: the defaults are on `posts."createdAt"`, `"updatedAt"`, `"publishedAt"`, `post_html_cache."renderedAt"`,
  and `sessions."createdAt"`. `CURRENT_TIMESTAMP` is in `CreateSession`, `TouchSession`, and `SavePostHTML`;
  `CreateDraft` relies on the `createdAt` default and passes `NULL` for `publishedAt` and `updatedAt` only because
  their defaults are the current time. After the change, those queries take the time as a parameter.
- Precision: Prisma wrote milliseconds; Go has written microseconds since cutover (seen locally:
  `2026-11-22 12:27:59.656983+01`). The SQLite format keeps milliseconds, so rounding in Go
  (`time.Now().Truncate(time.Millisecond)`, also on the times in the save transaction) makes the copy exact. The copy
  truncates the microsecond values written before the change: harmless.
- Title and slug: on the production copy, 442 titles and 516 slugs are `NULL`, none are `''`, and 86 posts have
  one without the other. Every query maps `NULL` to `''` (`COALESCE` in `Posts`, `Drafts`, `PostForEdit`) and back
  (`NULLIF` in `UpdatePost`), so Go never sees `NULL`. The migration: `UPDATE … SET title = '' WHERE title IS NULL`
  (and slug), then `SET DEFAULT ''`, `SET NOT NULL`. The seeds in `internal/db/db_test.go` and `tests/e2e/data.ts`
  insert `NULL`: switch them to `''`.
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
- Archive: replace the `EXTRACT` query with one returning `"publishedAt"` of published posts, newest first, and
  count months in Go in Europe/Rome (the prototype's `Archive`). Fixes the TODO in `site.sql` and `ARCHITECTURE.md`.
- Rename (optional): goose migration with `ALTER TABLE … RENAME COLUMN` for every camelCase column (`createdAt`,
  `updatedAt`, `publishedAt`, `hideTitle`, `bodyRevision`, `postId`, `tagName`, `lastActivityAt`,
  `rendererVersion`, `renderedAt`), and the `AS` aliases in the queries go away. Constraint names (`posts_number_key`,
  checked in `edit.go`) stay. A rollback to an older deployment would break, as with any migration.
- Skipped (2026-11-23): a unique index on `sessions.token`.

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
- sqlc already depends on v0.32.0 (in `go.sum` as indirect): Renovate will see two versions until sqlc updates.
- `ext/unicode`: Unicode-aware `upper`, `lower`, `LIKE` (`strings.EqualFold` rules, through a regexp cached per
  statement), `REGEXP`, collations, plus `casefold`, `unaccent`, `normalize`, `initcap`. Overriding `LIKE` turns off
  SQLite's LIKE optimization (indexes for prefix patterns): no query here uses it.
- FTS5 isn't compiled in: `ext/fts5.Register(conn)` adds it per connection, and allows tokenizers written in Go
  (an Italian stemmer, if word search ever needs it). `dbstat` isn't available.
- `_txlock` defaults to `immediate`. A busy timeout of 1 minute is set before any `_pragma`.
- `_timefmt` takes SQLite's formats; for formats without an offset (like `2006-01-02 15:04:05.000`,
  `TimeFormat4`) it converts to UTC before writing. Columns declared `DATETIME`, `DATE`, `TIME`, or `TIMESTAMP`
  scan into `time.Time`.
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
  and `STRICT` tables. The database is 1.5 MB, so the default 2 MB cache already holds all of it; `STRICT` conflicts
  with the declared types sqlc needs (see "Schema"). `PRAGMA optimize` is cheap: consider it.

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

The prototype's baseline (the real one also needs whatever revisions adds, if phase 2 lands first):

```sql
CREATE TABLE posts (
    id            INTEGER PRIMARY KEY,
    number        INTEGER  NOT NULL UNIQUE,
    category      TEXT     NOT NULL,
    created_at    DATETIME NOT NULL,
    updated_at    DATETIME,
    published_at  DATETIME,
    title         TEXT,
    slug          TEXT,
    hide_title    BOOLEAN  NOT NULL,
    body          TEXT     NOT NULL,
    body_revision INTEGER  NOT NULL DEFAULT 1
);
CREATE INDEX posts_published_at ON posts (published_at);
CREATE TABLE tags (name TEXT PRIMARY KEY) WITHOUT ROWID;
CREATE TABLE posts_tags (
    post_id  INTEGER NOT NULL REFERENCES posts (id) ON DELETE CASCADE ON UPDATE CASCADE,
    tag_name TEXT    NOT NULL REFERENCES tags (name) ON DELETE CASCADE ON UPDATE CASCADE,
    position INTEGER NOT NULL, -- the order typed in the editor (Step 0); the prototype didn't have it
    PRIMARY KEY (post_id, tag_name)
) WITHOUT ROWID;
CREATE INDEX posts_tags_tag_name ON posts_tags (tag_name);
CREATE TABLE sessions (
    id               INTEGER PRIMARY KEY,
    token            TEXT     NOT NULL UNIQUE,
    created_at       DATETIME NOT NULL,
    last_activity_at DATETIME NOT NULL
);
CREATE TABLE post_html_cache (
    post_id          INTEGER PRIMARY KEY REFERENCES posts (id) ON DELETE CASCADE ON UPDATE CASCADE,
    renderer_version INTEGER  NOT NULL,
    body_revision    INTEGER  NOT NULL,
    html             TEXT     NOT NULL,
    rendered_at      DATETIME NOT NULL
);
```

- Not `STRICT`: strict tables allow only `INT INTEGER REAL TEXT BLOB ANY`, and the declared `DATETIME` and
  `BOOLEAN` are what make sqlc generate `time.Time` and `bool`, and the driver decode them.
- No time defaults: `CURRENT_TIMESTAMP` writes `2026-11-22 10:00:00`, a different format from the driver's, and
  mixed formats break comparisons. Queries take the time as a parameter (`CreateSession`, `TouchSession`,
  `CreateDraft`, `SavePostHTML`). If a default is ever needed: `strftime('%Y-%m-%d %H:%M:%f', 'now')`.
- `sessions.token` gained `UNIQUE` (an index for the lookup; Postgres had none).
- Prisma's times have millisecond precision already, so the format loses nothing.
- Later schema changes: `ADD COLUMN` works; changing a column's type or constraints means rebuilding the table
  (create, copy, drop, rename), and goose's `-- +goose NO TRANSACTION` when foreign keys must be off.
- The goose history restarts at version 1 in the new file; the Postgres migrations go away with Postgres.

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
3. Weak inference: in `sqlc.narg(x) IS NULL OR col = sqlc.narg(x)` the parameter is `interface{}`.
   `CAST(sqlc.narg(x) AS TEXT) IS NULL` (or `AS INTEGER`) gives `*string` (`*int`). `NULLIF(@title, '')` needs
   `NULLIF(CAST(@title AS TEXT), '')`, and `json_each(@names)` needs `json_each(CAST(@names AS TEXT))`.
4. Never `CAST(x AS DATETIME)`: `DATETIME` has NUMERIC affinity, so `CAST('2026-11-22 10:00:00.000' AS DATETIME)`
   is the integer `2026`. The two date filters (`published_from`, `published_until`) stay `interface{}`, and the
   wrapper passes `time.Time` or nil.
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
| `::int`, `::text`, `::timestamptz` | removed, or `CAST` as in point 3 |
| `pgconn.PgError.ConstraintName` | `sqlite3.Error.ExtendedCode() == CONSTRAINT_UNIQUE` (the message names `posts.number`) |
| `pgx.ErrNoRows`, `pgx.BeginFunc` | `sql.ErrNoRows`, `BeginTx` + `defer Rollback` + `Commit` |

The `Posts` filter, as it generates:

```sql
WHERE p.published_at IS NOT NULL
  AND (CAST(sqlc.narg(number) AS INTEGER) IS NULL OR p.number = sqlc.narg(number))
  AND (CAST(sqlc.narg(category) AS TEXT) IS NULL OR p.category = sqlc.narg(category))
  AND (CAST(sqlc.narg(tag) AS TEXT) IS NULL OR EXISTS (
          SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name = sqlc.narg(tag)))
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
          OR (p.title LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR (p.body LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id
                     AND (pt.tag_name LIKE sqlc.narg(search_pattern) ESCAPE '\')))
  AND (sqlc.narg(published_from) IS NULL OR p.published_at >= sqlc.narg(published_from))
  AND (sqlc.narg(published_until) IS NULL OR p.published_at < sqlc.narg(published_until))
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

## Railway

- A service with a volume can't run two deployments at once: the old one stops before the new one starts, so each
  deploy has a short downtime (accepted). One volume per service, no replicas.
- Volumes are mounted as root, when the container starts (not at build or pre-deploy). The image runs as `nobody`.
  Railway's only documented fix is `RAILWAY_RUN_UID=0`, which runs the server as root. To keep `nobody`: start as
  root, `chown` the mount, and drop to `nobody` (Debian's `setpriv`, in util-linux). Decide when adding the volume.
- Railway's volume backups (manual or scheduled, incremental) can't be downloaded: fine as a second layer only.
- Today's backups: the private repo `notedimatteo-backups` runs `supabase db dump` daily against Postgres
  (`go-migration-notes.md`, "Production data"). A file inside the volume isn't reachable from GitHub Actions.
  Options:
  - The server runs `VACUUM INTO` (a consistent copy while running) daily and uploads it to R2 with the existing
    S3 client; the backups repo's Action downloads it and commits `sqlite3 .dump` output (text diffs as today).
  - Litestream streams WAL changes to R2 continuously (point-in-time restore), as a second process in the image.
- Getting the copy onto the volume, options: upload the file (`railway ssh`), or a one-off server mode that imports
  from Postgres at start when the file is empty (`IMPORT_DATABASE_URL`), removed after cutover.
- Rollback after cutover: Railway's previous deployment reads Postgres, which misses whatever was written to
  SQLite since. Copy it back by hand, or don't write until the new deploy looks right.

## Tests and tooling

- `internal/db` tests: create a temp file, migrate, seed (`testSeed` needs snake_case columns). `openTestTx` keeps
  working with `*sql.Tx` (sqlc's `DBTX` accepts it). For a production copy, copy the file and point the tests at it.
- `TestMigrationsLoad` opens `sql.Open("pgx", …)`: switch it to the SQLite driver.
- Browser tests: the global setup starts the server on a temp file (it migrates), then seeds with Node 26's built-in
  `node:sqlite` (`DatabaseSync`, SQLite 3.54.1; checked). WAL and the busy timeout let it write while the server
  runs. `@testcontainers/postgresql` goes; rclone keeps Docker.
- The local `sqlite3` (macOS, 3.54.0) opens the files for ad-hoc queries.
- Renovate: drop the `postgres:18` rule. Both drivers released within days of 2026-11-22; the 3-day minimum age
  applies.
