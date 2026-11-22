# Go migration plan

Replace the persistent SvelteKit/Node server with a Go server that renders HTML,
while keeping the interactive editor in Svelte 5. The aim is lower, stable memory
usage. Done: Go has served production since 2026-11-21. The redesign continues in
`redesign-plan.md`.

Working notes and gotchas: `go-migration-notes.md`.

## Decisions

| Area           | Choice                                                                                                                                                            |
|----------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Routing        | `net/http` `ServeMux`; one `/{seg}` handler separates post numbers from categories                                                                                |
| URLs           | Built only in `internal/urls`; site address from config                                                                                                              |
| Templates      | templ components in `internal/views` with typed parameters; generated `_templ.go` files are committed, and CI checks formatting and generation                    |
| Database       | `pgx/v5` with sqlc: SQL in `internal/db/queries`, typed Go code generated and checked against the schema; small pool                                              |
| Migrations     | `goose` from step 7, with the existing schema as baseline; run at server start since cutover                                                                                       |
| Markdown       | goldmark in its own package, with options per content type; review output differences on real posts during cutover                                                |
| Images         | `govips` (cgo) against sharp's prebuilt libvips from npm, with mozjpeg; libvips caching turned off                                                               |
| Sessions       | Keep the existing table and cookie name                                                                                                                           |
| CSRF           | `http.CrossOriginProtection`                                                                                                                                      |
| Object storage | `aws-sdk-go-v2` S3 client                                                                                                                                         |
| Editor data    | JSON in `<script type="application/json">`                                                                                                                        |
| Editor saves   | `fetch` with `FormData`, returning JSON field errors; validation lives only in Go                                                                                 |
| Assets         | Vite manifest with hashed files, embedded in the binary                                                                                                           |
| CSS            | Tailwind v4, scanning `internal/views`                                                                                                                           |
| Repo layout    | Go module at the root: `internal/server` (routes, handlers), `internal/views` (templ), `internal/urls` (URL builders); the editor in `web/editor`, copied from SvelteKit |
| Deployment     | Multi-stage Dockerfile on Railway, with sharp's libvips from npm at build time and runtime                                                                       |
| Observability  | Drop OpenTelemetry; use `slog` and authenticated `pprof`                                                                                                          |
| Dates          | Embed `time/tzdata` and hard-code Italian month names                                                                                                             |

## Architecture

- Go renders posts, categories, archives, search, pagination, login, and draft pages.
- Handlers pass plain data to templates and never build markup, so a redesign only
  touches templates and CSS.
- Ordinary links and HTML forms handle navigation, search, authentication, and deletion.
- Small browser scripts handle theme selection, sidebar tag sorting, and delete confirmation.
- A standalone Svelte editor mounts inside the Go-rendered edit page. Go supplies
  safely encoded initial post data, available tags, and endpoint URLs.
- Go serves the compiled JavaScript and CSS. Node, Vite, Svelte, and Tailwind remain
  build tools; production runs Go without a Node server.
- Keep PostgreSQL, the existing schema, and S3-compatible image storage during the migration.
  Replace Prisma access with `pgx`, without combining this with a database migration.

## Editor integration

Keep Markdown shortcuts, cursor handling, textarea resizing, tag suggestions,
slug normalization, pasted-image uploads, preview, dirty tracking, save feedback,
and publish/unpublish controls.

Reuse the existing TypeScript editor helpers and Svelte components where practical.
Replace SvelteKit's `$app` dependencies with props, and replace Superforms' action
integration with local Svelte state and `fetch` requests.

| Endpoint responsibility  | Go behavior                                                                                  |
|--------------------------|----------------------------------------------------------------------------------------------|
| Save, publish, unpublish | Validate input, update posts/tags/HTML cache transactionally, return success or field errors |
| Preview                  | Render Markdown using the same renderer as published posts                                   |
| Upload                   | Process original and preview images, upload them, return paths and dimensions                |

Keep requests on the same origin with session-cookie authentication. Enforce
authorization, CSRF protection, and input validation in Go.

## Steps

Go has served production since 2026-11-21 (step 9); SvelteKit served it until then, so nothing partial reached visitors. There's no second
Railway service: the Go server replaces SvelteKit on the same one.

### Step 1: Skeleton

- [x] Create the `go-migration` branch from `main` and commit the plan documents
- [x] Add the Go module, `cmd/server` entry point, and environment config
- [x] Add the `net/http` server with logging, error handling, and graceful shutdown
- [x] Connect to the existing local database with a small `pgx` pool
- [x] Add a separate Vite build for CSS and fonts, with hashed files and a manifest
- [x] Embed built assets in the binary and serve them with long-lived cache headers
- [x] Parse embedded templates at startup and render the shared layout
- [x] Render the home page with real posts, using the cached post HTML
- [x] Add the Dockerfile with libvips at build time and runtime, and confirm it builds
- [x] Add Go formatting, `go vet`, tests, and the asset build to CI

### Step 2: Public site

- [x] Build post URLs in one Go function, and read the site address from config
- [x] Port shared layout parts: top bar, header, footer, sidebar, and metadata
- [x] Port the theme script and sidebar tag sorting, keeping saved theme preferences
- [x] Port post queries and shared post rendering, including Europe/Rome dates
- [x] Port pagination
- [x] Add goldmark as its own package with options per content type; notes get strikethrough and no raw HTML
- [x] Add `[img]` handling, preserving existing image URLs
- [x] Add HTML cache reads and writes, with a new renderer version, replacing the temporary `<pre>` fallback
- [x] Port the home and single post pages, including slug behavior
- [x] Port the category pages
- [x] Port the tag and year/month archive pages
- [x] Port search
- [x] Port the about page
- [x] Port the RSS feed
- [x] Port 404 and error pages, and match trailing-slash behavior
- [x] Add a status test for every public page, and function or component tests for page logic

### Step 3: Authentication and draft management

- [x] Port login and logout, keeping the session table and cookie name
- [x] Port session loading and the `lastActivityAt` update
- [x] Add CSRF protection to state-changing requests
- [x] Show admin-only controls on Go-rendered pages for logged-in users
- [x] Port the unpublished list, including relative dates
- [x] Port new draft creation
- [x] Port post deletion and its confirmation script
- [x] Add tests for logged-out access to protected routes and actions

### Step 4: Editor backend

- [x] Port post form validation and field errors
- [x] Port the save transaction: post, tags, and HTML cache
- [x] Port the publish and unpublish behavior
- [x] Add the JSON response format for success and field errors
- [x] Port the Markdown preview endpoint
- [x] Decide on JPEG encoding: mozjpeg, through sharp's prebuilt libvips from npm instead of Debian's
- [x] Add `govips` startup with caching turned off, and remove `cmd/vipscheck`
- [x] Port image processing: orientation, JPEG output, and 1200px previews
- [x] Add S3 uploads, preserving object keys and the `original` metadata
- [x] Port the upload endpoint with a request size limit
- [x] Add tests for validation, the save transaction, and image processing
- [x] Test save, publish, preview, and upload against local services without the UI

### Step 5: Standalone editor

- [x] Add a Vite entry that mounts the Svelte editor
- [x] Render the edit page in Go with safely encoded post data, tags, and endpoint URLs
- [x] Replace `$app` imports with props
- [x] Replace Superforms with local state, dirty tracking, and `fetch`
- [x] Connect field errors, save feedback, and keyboard shortcuts
- [x] Connect preview, tag suggestions, slug normalization, and pasted-image uploads
- [x] Copy the editor helpers and their tests into `web/editor`, so it imports nothing from `src`
- [x] Check every editor feature in the browser against the Go endpoints

### Step 6: Removal

- [x] Review real posts for Markdown differences and fix the ones that matter
- [x] Check URLs, RSS, login, and an upload against a copy of production data
- [x] Review the Go code against SvelteKit's, file by file, for behavior lost by accident
- [x] Remove SvelteKit, Prisma, OpenTelemetry, and unused Node dependencies
- [x] Update CI, `package.json`, `README.md`, and `AGENTS.md`
- [x] Remove mentions of SvelteKit from Go, TypeScript, and template comments ("like the SvelteKit app"): keep only what still explains the code

### Step 7: Goose

- [x] Add `goose` with the existing schema as the baseline migration
- [x] Point `sqlc.yaml` at the goose migrations
- [x] Keep `_prisma_migrations`, and don't run migrations at server start, so a rollback still works

### Step 8: Browser tests

- [x] Add Playwright, with a global setup that builds and starts the Go server
- [x] Start Postgres and rclone's S3 with testcontainers; apply the goose migrations and a small seed
- [x] Test the main flows: reading, login and logout, writing a note with an upload, and editor errors
- [x] Run them in CI, keeping the Playwright report when they fail

### Scripts review (after step 8)

- [x] One npm script per purpose: `generate`, `format`, `check`, `test`; `dev`, `check`, and the Go and browser tests generate first
- [x] Write `.libvips` in `postinstall`
- [x] CI jobs `check`, `test` (a matrix over `test:js`, `test:go`, `test:e2e`), and `docker` run the same scripts

### Step 9: Cutover

- [x] Deploy the Go server to the existing Railway service (2026-11-21); Railway's previous deployment is the rollback
- [x] Watch logs and Railway's memory graph after the switch
- [x] Drop `_prisma_migrations` in a goose migration (`00002`)
- [x] Run the migrations at server start; remove `cmd/migrate`, `npm run migrate`, and the e2e `migrate` binary

Completion means feature parity, preserved URLs/data, reviewed Markdown and image
output, working editor errors and save/publish flows, and a production artifact
that needs no Node server. Run focused Go tests, existing editor helper tests, and
the required formatting, lint, and Svelte checks.

### After cutover

- [x] Pin GitHub Actions to exact versions with `actions.lock` (`gh actions-lock`), and add `.nvmrc`
- [x] Add Renovate for npm, Go modules, the Dockerfile, and GitHub Actions; enabling the Renovate app is left
- [x] Move the lasting facts from the notes into `docs/ARCHITECTURE.md`; keep the plan and notes in `docs/`
