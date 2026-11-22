# Architecture

How the site works and why it's built this way. `README.md` covers setup and the npm scripts.

## Overview

- One Go binary (`cmd/server`) serves the whole site. It renders pages with [templ](https://templ.guide), reads
  and writes PostgreSQL, stores images in an S3-compatible bucket (Cloudflare R2), and processes them with libvips.
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
| `internal/db` | pgx pool, sqlc queries (`queries/`), goose migrations (`migrations/`) |
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

- Queries live in `internal/db/queries/*.sql`; sqlc generates typed Go code from them and from the migrations.
  The generated files are committed. Conventions:
  - Overrides map `int4`/`serial` to `int` and `timestamptz` to `time.Time`. Cast integer parameters and
    results to `::int`.
  - Filters are optional parameters (`sqlc.narg(x) IS NULL OR ...`), so the query text never changes. `Posts`
    and `CountPosts` repeat the same `WHERE`; a test checks they agree.
  - Row types like `db.Post` are aliases of sqlc's generated structs.
- The schema comes from the SvelteKit app's Prisma migrations, so columns are camelCase (`"publishedAt"`).
- goose migrations are embedded and run when the server starts. `00001_baseline.sql` recreates the Prisma
  schema with `IF NOT EXISTS`, so on the production database it only recorded version 1. Migrations have no Down.
- Production Postgres runs in UTC; archive months group by the database session's time zone (a TODO).

## Markdown

- goldmark with strikethrough; raw HTML is dropped (inline tags removed, text kept).
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
  `SameSite=Lax`, and `Secure` except on localhost. `lastActivityAt` updates at most once a minute.
- Logout, new draft, and delete are POST forms, because Lax cookies reach GET links from other sites.
- TODOs: expire sessions in the database, and limit failed logins.

## Editor

- `GET /{number}/edit` renders the layout with the editor's data in `<script type="application/json">`
  (`templ.JSONScript`) and the editor bundle. The data includes each post's URLs, built by `internal/urls`.
- The editor saves with `fetch` and `FormData` to `POST /{number}/edit`, which validates in Go and answers
  `{"post": …}` or 400 with `{"errors": {"field": "message"}}`. Validation lives only in Go.
- `POST /{number}/edit/preview` renders Markdown with the same renderer as published posts.
- `form.ts` tracks unsaved changes against what the last successful save sent.
- The save locks the post's row and updates the post, its cached HTML, and its tags in one transaction.

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
- Database tests (`internal/db`) start `postgres:18` with testcontainers-go, apply the migrations, and load a
  small seed. `-short` skips them; `TEST_DATABASE_URL` runs them against another database as it is, and tests
  that write roll back.
- Browser tests (`tests/e2e`) build the server, start `postgres:18` and rclone's S3 server with testcontainers,
  start the server on port 6210 (which migrates), load the seed from `data.ts`, and run Playwright. A test fails
  on any uncaught JavaScript error, and requests to other hosts are aborted.
- `docker build --target libvips-test .` runs the image processing tests inside the production image.

## CI and deployment

- GitHub Actions jobs: `check` (formatting, lint, types, vet, generated code committed), `test` (a matrix over
  `test:js`, `test:go`, `test:e2e`), `docker`, and `verify-lockfile`. Actions use exact versions, locked in
  `.github/workflows/actions.lock`: run `gh actions-lock` after changing a `uses:`.
- The Dockerfile has stages `assets` (Node: npm, libvips, Vite) → `build` (Go) → `runtime` (Debian trixie with
  sharp's `libvips-cpp.so`) → `server`, about 146 MB, running as `nobody`.
- Railway builds the Dockerfile from `main` and runs the `server` stage. Railway's previous deployment is the
  rollback.
- Renovate (`renovate.json`): packages that run on the server (Go modules, the Go version, libvips) weekly on
  Wednesdays; build and browser tooling monthly on the 15th; a 3-day minimum release age, which npm also
  enforces through `.npmrc`. templ and sqlc updates need `npm run generate` pushed by hand.

## Configuration

| Variable | |
|---|---|
| `DATABASE_URL` | Required |
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | Required |
| `STORAGE_PUBLIC_BASE_URL` | Required: the public address of uploaded images |
| `STORAGE_S3_ENDPOINT`, `_REGION`, `_ACCESS_KEY_ID`, `_SECRET_ACCESS_KEY`, `_BUCKET` | All or none; without them, uploads are off |
| `SITE_URL` | Default `https://notedimatteo.it`; GoatCounter runs only on that host |
| `PORT` | Default 6200 |
