# Go migration plan

Replace the persistent SvelteKit/Node server with a Go server that renders HTML,
while keeping the interactive editor in Svelte 5. The aim is lower, stable memory
usage. Later phases merge the blog (matteosonoio.it, Hugo) into the same site.

Working notes, gotchas, and open questions: `GO-MIGRATION-NOTES.md`.

## Phases

Each phase reaches production on its own. Phases 3–5 can happen in any order.

1. **Go migration:** same features and look as today, then cutover. Detailed below.
2. **Revisions:** keep every body revision of a post.
3. **New design:** the notes site in the new design, changing only templates and CSS.
4. **Articles:** import the blog, bilingual pages, domains and redirects.
5. **New editor:** CodeMirror 6 with live preview.

Design reference: the [design canvas](https://claude.ai/artifact/DeA9bBLoajjHfTZfgk1uU2),
boards E2/E4 (home), D2 (notes), D3 (article), D4 (article archive), F2 (projects),
and G1–G3 (editor).

## Decisions

| Area           | Choice                                                                                                                                                            |
|----------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Routing        | `net/http` `ServeMux`; one `/{seg}` handler separates post numbers from categories                                                                                |
| URLs           | Built only in `internal/urls`; site address from config; the merged site's URL scheme is decided in phase 4                                                       |
| Templates      | templ components in `internal/views` with typed parameters; generated `_templ.go` files are committed, and CI checks formatting and generation                    |
| Database       | `pgx/v5`, hand-written SQL, small pool                                                                                                                            |
| Migrations     | None in phase 1; `goose` from phase 2, with the existing schema as baseline                                                                                       |
| Markdown       | goldmark in its own package, with options per content type; review output differences on real posts during cutover                                                |
| Images         | `govips` (cgo) with libvips caching turned off                                                                                                                    |
| Sessions       | Keep the existing table and cookie name                                                                                                                           |
| CSRF           | `http.CrossOriginProtection`                                                                                                                                      |
| Object storage | `aws-sdk-go-v2` S3 client                                                                                                                                         |
| Editor data    | JSON in `<script type="application/json">`                                                                                                                        |
| Editor saves   | `fetch` with `FormData`, returning JSON field errors; validation lives only in Go                                                                                 |
| Assets         | Vite manifest with hashed files, embedded in the binary                                                                                                           |
| CSS            | Tailwind v4, scanning `internal/views`; from phase 3, only design tokens, no arbitrary values, and a Go test that checks component classes                        |
| Repo layout    | Go module at the root: `internal/server` (routes, handlers), `internal/views` (templ), `internal/urls` (URL builders); editor helpers stay in place until cutover |
| Deployment     | Multi-stage Dockerfile on Railway, with libvips at build time and runtime                                                                                         |
| Observability  | Drop OpenTelemetry; use `slog` and authenticated `pprof`                                                                                                          |
| Dates          | Embed `time/tzdata` and hard-code Italian month names; English ones in phase 4                                                                                    |

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
- Keep PostgreSQL, the existing schema, and S3-compatible image storage in phase 1.
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

## Phase 1: Go migration

SvelteKit keeps serving production until step 6, so nothing partial reaches visitors.

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

- [ ] Port login and logout, keeping the session table and cookie name
- [ ] Port session loading and the `lastActivityAt` update
- [ ] Add CSRF protection to state-changing requests
- [ ] Show admin-only controls on Go-rendered pages for logged-in users
- [ ] Port the unpublished list, including relative dates
- [ ] Port new draft creation
- [ ] Port post deletion and its confirmation script
- [ ] Add tests for logged-out access to protected routes and actions

### Step 4: Editor backend

- [ ] Port post form validation and field errors
- [ ] Port the save transaction: post, tags, and HTML cache
- [ ] Port the publish and unpublish behavior
- [ ] Add the JSON response format for success and field errors
- [ ] Port the Markdown preview endpoint
- [ ] Decide on JPEG encoding: Debian's libvips ignores the mozjpeg options, so either accept larger files or build libvips with mozjpeg
- [ ] Add `govips` startup with caching turned off, and remove `cmd/vipscheck`
- [ ] Port image processing: orientation, JPEG output, and 1200px previews
- [ ] Add S3 uploads, preserving object keys and the `original` metadata
- [ ] Port the upload endpoint with a request size limit
- [ ] Add tests for validation, the save transaction, and image processing
- [ ] Test save, publish, preview, and upload against local services without the UI

### Step 5: Standalone editor

- [ ] Add a Vite entry that mounts the Svelte editor
- [ ] Render the edit page in Go with safely encoded post data, tags, and endpoint URLs
- [ ] Replace `$app` imports with props
- [ ] Replace Superforms with local state, dirty tracking, and `fetch`
- [ ] Connect field errors, save feedback, and keyboard shortcuts
- [ ] Connect preview, tag suggestions, slug normalization, and pasted-image uploads
- [ ] Run the existing editor helper tests against the new setup
- [ ] Check every editor feature in the browser against the Go endpoints

### Step 6: Cutover and cleanup

- [ ] Review real posts for Markdown differences and fix the ones that matter
- [ ] Check URLs, RSS, login, and an upload against a copy of production data
- [ ] Deploy the Go service to Railway without production traffic
- [ ] Switch production traffic, keeping the old deployment ready for rollback
- [ ] Watch logs and Railway's memory graph after the switch
- [ ] Remove SvelteKit, Prisma, OpenTelemetry, and unused Node dependencies
- [ ] Update CI, `package.json`, `README.md`, and `AGENTS.md`

Completion means feature parity, preserved URLs/data, reviewed Markdown and image
output, working editor errors and save/publish flows, and a production artifact
that needs no Node server. Run focused Go tests, existing editor helper tests, and
the required formatting, lint, and Svelte checks.

## Phase 2: Revisions

- [ ] Add `goose` with the existing schema as baseline, and drop `_prisma_migrations`
- [ ] Add a revisions table keyed by post and `bodyRevision`
- [ ] Save a revision in the save transaction whenever the body changes
- [ ] List revisions on the edit page and restore one

## Phase 3: New design

- [ ] Reset the Tailwind theme to the design tokens: colors, fonts, and spacing
- [ ] Self-host Public Sans and Source Serif 4
- [ ] Add a Go test that fails on arbitrary values and on template classes missing from the built CSS
- [ ] Move repeated patterns into template partials or `@utility` rules
- [ ] Document the tokens and partials in `AGENTS.md`
- [ ] Redesign the layout, home, notes, archive, search, and about pages; the home has no articles until phase 4

## Phase 4: Articles

- [ ] Decide the merged site's URL scheme and domains
- [ ] Add article fields: type, language, summary, last-updated date, table of contents and comments flags
- [ ] Import the Hugo posts: front matter, `fig`/`relref`/`center` shortcodes, raw HTML (`<sup>`, `<video>`)
- [ ] Upload the blog's images and videos to S3, with WebP versions
- [ ] Enable raw HTML, Chroma syntax highlighting, footnotes, and tables for articles
- [ ] Add article, article archive, and projects pages
- [ ] Add Italian/English interface strings, English dates, and `hreflang` links; notes stay Italian-only
- [ ] Add related notes on articles, by tag
- [ ] Add giscus comments, keeping article paths so the `pathname` mapping still finds threads
- [ ] Add feeds for notes, articles, and everything
- [ ] Redirect old blog URLs, Hugo feeds, and `blog.contrini.it`
- [ ] Retire Hugo and Netlify

## Phase 5: New editor

- [ ] Replace the textarea with CodeMirror 6 live preview, still storing plain Markdown
- [ ] Add the quick composer on the notes page (G1)
- [ ] Add the full editor with the details sidebar (G2)
- [ ] Add the drafts page (G3)
- [ ] Add the `/` menu, moving blocks, and image paste and drag
- [ ] Fetch page titles for pasted links, refusing private network addresses
- [ ] Add note links (`#670`) and footnotes to the renderer
