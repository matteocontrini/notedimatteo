# Architecture

How the site works and why it's built this way. `README.md` covers setup and the npm scripts.

## Overview

- One Go binary (`cmd/server`) serves the whole site. It renders pages with [templ](https://templ.guide), reads
  and writes a SQLite file, stores images in an S3-compatible bucket (Cloudflare R2), and processes them with libvips.
- The post editor is a Svelte 5 app that Go mounts on the edit page. Everything else is plain HTML: links and
  forms handle navigation, search, login, and deletion. Small scripts handle the theme, sidebar tag sorting, and
  delete confirmation.
- Vite builds the CSS (Tailwind), the site script, and the editor. The binary embeds the result, so production
  runs Go alone, without Node.
- The server uses about 16 MB of memory in production (SvelteKit, before November 2026: 500 MB or more).

## Code layout

| Path | Contents |
|---|---|
| `cmd/server` | Entry point: configuration, migrations, libvips, the HTTP server, graceful shutdown |
| `internal/config` | Environment variables |
| `internal/server` | Routes, middleware, handlers; builds the data each page shows |
| `internal/views` | templ components, date formatting (Europe/Rome, Italian month names) |
| `internal/urls` | Every URL the site builds |
| `internal/db` | SQLite connections, sqlc queries (`queries/`), goose migrations (`migrations/`) |
| `internal/markdown` | goldmark renderer and the `[img]` syntax |
| `internal/images` | Upload flow: object keys and the two uploads (plain Go) |
| `internal/images/libvips` | Image processing with govips (cgo) |
| `internal/storage` | S3 client |
| `web/assets`, `web/editor` | CSS, site script, and the Svelte editor; Vite writes `web/dist`, which `web` embeds |
| `tests/e2e` | Playwright browser tests |

Handlers load data and pass it to components as typed parameters. Components never query the database and
never build URLs by hand, so a redesign only touches `internal/views` and the CSS.

## Requests

- Routes use `net/http`'s `ServeMux` (`internal/server/server.go`). `/{segment}` serves both post numbers and
  categories, and `/{number}/{slug}` also serves `/{number}/edit`: separate patterns would overlap, and
  `ServeMux` panics on overlapping patterns.
- Every request passes through, in order: logging, panic recovery, a 308 that removes a trailing slash, and
  `http.CrossOriginProtection`, which answers POSTs from other sites with a 403.
- `handle` loads the session before each page handler, and only queries the database when the request has the
  session cookie. Editor endpoints use `handleAPI` instead: errors are JSON, and logged-out requests get 401.
- `render` renders a page into a buffer first, so an error shows a clean error page. Pages get `Content-Length`
  (so the CDN compresses them) and `Cache-Control: no-cache`, or `private, no-cache` for a logged-in user,
  whose pages show admin controls. There are no ETags.
- Assets have hashed names and get `immutable` for a year; the favicon gets one day; the feed one minute.
- `GET /healthz` checks the database.

## Database

- SQLite through [ncruces/go-sqlite3](https://github.com/ncruces/go-sqlite3) (without cgo) and `database/sql`,
  in one file (`DATABASE_PATH`). Every connection uses WAL, `foreign_keys` (deletes cascade only with it), a busy
  timeout, and immediate transactions, which take the write lock when they begin. Two connections: one can read
  while the other writes.
- `ext/unicode` replaces `LIKE`, so search ignores the case of every letter, accented ones included.
- Queries live in `internal/db/queries/*.sql`; sqlc generates typed Go code from them and from the migrations.
  The generated files are committed. Conventions:
  - Overrides map `INTEGER` and `integer` to `int` (sqlc matches the case), each time column to `db.Time`, and
    `posts.hide_title` to `bool`. A new time or boolean column needs its own override.
  - Filters are optional parameters (`col = sqlc.narg(x) OR sqlc.narg(x) IS NULL`), so the query text never
    changes. sqlc types a parameter from its first use, so the comparison comes first. `Posts` and `CountPosts`
    repeat the same `WHERE`; a test checks they agree.
  - Tags come as a JSON array (`json_group_array`), which Go decodes into `db.Post` and `db.EditablePost`; saves
    pass tags as a JSON array to `json_each`.
- Tables are `STRICT`: a value of the wrong type is an error. `STRICT` has no time or boolean types:
  - Times are `TEXT` in UTC with milliseconds (`2026-11-23 10:00:00.000`), the format of SQLite's date
    functions, so comparing and sorting the text compares the times. A `CHECK` on each column rejects any other
    format; `TestTimeColumns` checks that every `*_at` column has it. `db.Time` writes and reads the format, and
    the wrappers return `time.Time`. Go sets every time (`currentTime` in `internal/db`); no column has a time
    default.
  - Booleans are `INTEGER` with `CHECK (col IN (0, 1))`.
- goose migrations are embedded and run when the server starts. `00001_baseline.sql` is the schema moved from
  PostgreSQL, the SvelteKit app's database. Migrations have no Down.
- Archive months are in Europe/Rome: Go counts the publication times per month (`db.Archive`), and the month
  pages select from midnight in Rome to midnight in Rome.
- A post's tags keep the order they were typed in: `posts_tags.position`, numbered from 0 by each save.

## Markdown

- goldmark with strikethrough; HTML isn't parsed, so tags like `<details>` show as text.
- `[img width=… height=…]path[/img]` shows the `_preview` variant of an uploaded image, lazy-loaded and linked
  to the original (or to `link`, which must be http(s) or a path on the site).
- Rendered HTML is cached in `post_html_cache` with `markdown.Version`. Bump the version whenever the output
  bytes change: every post then re-renders once, on its next view.
- `testdata/golden/*.md` and their expected `.html` are compared byte for byte; `go test ./internal/markdown
  -update` rewrites the expected output after an intended change.

## Images and uploads

- `POST /{number}/edit/upload` takes a `file` up to 20 MB and answers `{main, preview, width, height}`. A file
  libvips can't read is a 400; without the `STORAGE_S3_*` variables, uploads answer 503.
- Processing: auto-orientation, conversion to sRGB (then the ICC profile is stripped), a JPEG at quality 85
  and a 1200 px wide preview at quality 80, both with mozjpeg.
- Keys are `YYYY/MM/<12 hex>.jpg` and `…_preview.jpg`. The S3 metadata `original` holds the uploaded file name,
  RFC 2047-encoded when it isn't ASCII.
- An upload continues if the client goes away, so the main image is never stored without its preview.

### libvips

- libvips is sharp's prebuilt build from npm (`@img/sharp-libvips-*`): one library with every dependency built
  in, with mozjpeg. The same version runs in development and production.
- `npm install` runs `scripts/libvips.mjs`, which writes `.libvips/vips.pc`. Go commands that compile
  `internal/images/libvips` or `cmd/server` need `PKG_CONFIG_LIBDIR="$PWD/.libvips"` (an absolute path); the npm
  scripts set it, and the editor's Go settings need it too.
- `magick.c` stubs `vips_magickload_buffer`, which sharp's build lacks; formats like BMP therefore fail to load.
- libvips caching is off (`vips.Startup` with a zero config).
- Rejected alternatives: Debian's libvips (no mozjpeg: files 14–19% bigger), Homebrew's `vips` (28 dependencies,
  ImageMagick and Poppler included), and development inside Docker (the image would still need sharp's build).

## Authentication

- One admin account from `ADMIN_USERNAME` and `ADMIN_PASSWORD`, compared in constant time.
- Sessions live in the `sessions` table; the cookie is `notedimatteo_session`: 60 days, `HttpOnly`,
  `SameSite=Lax`, and `Secure` except on localhost. `last_activity_at` updates at most once a minute.
- Logout, new draft, and delete are POST forms, because Lax cookies reach GET links from other sites.
- TODOs: expire sessions in the database, and limit failed logins.

## Editor

- `GET /{number}/edit` renders the layout with the editor's data in `<script type="application/json">`
  (`templ.JSONScript`) and the editor bundle. The data includes each post's URLs, built by `internal/urls`,
  and the public address of uploaded images.
- The body field is CodeMirror 6 with the CommonMark parser and strikethrough, in `web/editor/body`: one module
  per part (`language`, `theme`, `formatting`, `blocks` for quotes and code, `paragraphs`, `paste`, `images`,
  `upload-placeholders`), assembled in `index.ts`, each with its tests. The text stays plain Markdown: the editor
  styles it like a published post, dims the markers, shows each `[img]` image below its line, and hides the
  blank lines between blocks, showing the space between paragraphs instead; Enter starts a paragraph. Pasted
  formatted text becomes Markdown (Turndown). Colors
  come from `--editor-*` variables in `web/assets/main.css`, light and dark.
- The editor saves with `fetch` and `FormData` to `POST /{number}/edit`, which validates in Go and answers
  `{"post": …}` or 400 with `{"errors": {"field": "message"}}`. Validation lives only in Go. Browsers send
  form text with `\r\n` line breaks; the endpoint stores `\n`.
- `POST /{number}/edit/preview` renders Markdown with the same renderer as published posts.
- `form.ts` tracks unsaved changes against what the last successful save sent.
- The save updates the post, its cached HTML, and its tags in one transaction, which holds the write lock from
  its start, so two saves never overlap.

## Assets

- `vite.config.ts` builds three entries from `web` (`main.css`, `site.ts`, `editor/main.ts`) into `web/dist`
  with a manifest. The server looks entries up in the manifest and refuses entries split into other files:
  pages load each entry with one tag.
- Tailwind scans `internal/views` and `web/editor` for class names.
- `go build`, `go test`, and `go vet` need `web/dist`: `npm run generate` builds it.

## Tests

- Go: logic is tested as plain functions or by rendering one component; HTTP tests check status codes, headers,
  and errors, with a fake store. templ's compiler catches missing fields, so no test only checks that a field
  reaches the page.
- Database tests (`internal/db`) build a template file with the migrations and a small seed, and each test opens
  its own copy. `TEST_DATABASE_PATH` replaces the seed with a copy of another database file, such as a copy of
  production data, which the tests never change.
- Browser tests (`tests/e2e`) build the server, start rclone's S3 server with testcontainers, start the server on
  port 6210 with a database file in a temporary directory (the server creates and migrates it), load the seed
  from `data.ts` with `node:sqlite`, and run Playwright. A test fails
  on any uncaught JavaScript error, and requests to other hosts are aborted.
- `docker build --target libvips-test .` runs the image processing tests inside the production image.

## CI and deployment

- GitHub Actions jobs: `check` (formatting, lint, types, vet, generated code committed), `test` (a matrix over
  `test:js`, `test:go`, `test:e2e`), `docker`, and `verify-lockfile`; then the Deploy workflow (below). Actions use exact versions, locked in
  `.github/workflows/actions.lock`: run `gh actions-lock` after changing a `uses:`.
- The Dockerfile has stages `assets` (Node: npm, libvips, Vite) → `build` (Go) → `runtime` (Debian trixie with
  sharp's `libvips-cpp.so`) → `server`, about 200 MB.
- The `server` stage runs Litestream (`litestream.yml`), which starts the server and copies every change of the
  database to a private bucket within a second. When the database file doesn't exist, it restores the latest copy
  first. Its binary comes from the `litestream/litestream` image.
- Railway builds the Dockerfile from `main` once Build passes (the service's "Wait for CI") and runs the `server`
  stage, with a volume at `/data`. It runs as root, because Railway mounts volumes as owned by root.
- Run one instance only, with no replicas: a second one would replicate another database to the same bucket path.
  With a volume, Railway stops the old deployment before starting the new one, so each deploy has a few seconds of
  downtime. SIGTERM goes to Litestream, which stops the server and makes a last sync:
  `RAILWAY_DEPLOYMENT_DRAINING_SECONDS=30` gives it 30 s before SIGKILL (Railway's default is 0).
- The bucket keeps 7 days of changes, restorable to any point. For a longer history, the private repo
  `notedimatteo-backups` restores the latest copy daily with a read-only token and commits a text dump
  (`notedimatteo.sql`, without sessions and cached HTML).
- The Build workflow's `docker` job also pushes the image to GHCR on `main`
  (`ghcr.io/matteocontrini/notedimatteo:<commit>`), and the Deploy workflow (`deploy.yml`, triggered by
  `workflow_run`) can deploy it to Bunny Magic Containers with `.github/scripts/deploy.js`. It's off while the
  `BUNNY_APP_ID` repository variable isn't set: Bunny's volumes don't let two processes share a SQLite database
  (`sqlite-notes.md`).
- Renovate (`renovate.json`): packages that run on the server (Go modules, the Go version, libvips) weekly on
  Wednesdays; build and browser tooling monthly on the 15th; a 3-day minimum release age, which npm also
  enforces through `.npmrc`. templ and sqlc updates need `npm run generate` pushed by hand.

## Configuration

| Variable | |
|---|---|
| `DATABASE_PATH` | Required: the SQLite file, created if missing |
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | Required |
| `STORAGE_PUBLIC_BASE_URL` | Required: the public address of uploaded images |
| `STORAGE_S3_ENDPOINT`, `_REGION`, `_ACCESS_KEY_ID`, `_SECRET_ACCESS_KEY`, `_BUCKET` | All or none; without them, uploads are off |
| `SITE_URL` | Default `https://notedimatteo.it`; GoatCounter runs only on that host |
| `PORT` | Default 6200 |
| `LITESTREAM_ENDPOINT`, `_BUCKET`, `_ACCESS_KEY_ID`, `_SECRET_ACCESS_KEY` | Required by the Docker image: the bucket for the database's backups (`litestream.yml`) |
