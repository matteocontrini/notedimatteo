# Go migration notes

Working notes for `GO-MIGRATION-PLAN.md`: facts found while working, gotchas, and
open questions. Read both before resuming. Record what the code and git history
don't show; remove notes that stop being true.

## Working agreements

- Phase 1 parity means the same pages, URLs, and behavior for what visitors and existing links produce. Edge-case
  input handling (number parsing, odd paths, error wording) uses plain, idiomatic Go instead of emulating SvelteKit.
  Visible fixes that change real content are deferred as `TODO` comments (`grep -rn TODO cmd internal web`).
- The user handles git: never commit, push, or stage.
- Plan terminology: phases are the broad units, steps are the smaller ones inside a phase.
- Keep plan and notes succinct and schematic.

## Open questions

- Merged site's URL scheme and domains (phase 4). Affects routes, redirects, and giscus threads.
- English home: a variant of board E3, not designed yet. Italian home: E2 or E4.

## Local development

- SvelteKit: `npm run dev`, port 6100. Go: `npm run go:dev`, port 6200 (builds assets, sources `.env`, `go run`).
- No hot reload: restart `go:dev` after template or CSS changes. `go:dev` runs `go tool templ generate` first.
- templ is pinned with the `tool` directive in `go.mod`: run it as `go tool templ` (`generate`, `fmt`, `lsp` for editors).
  Generated `*_templ.go` files are committed; after editing a `.templ` file, run `go tool templ generate` and `templ fmt`.
- `templ fmt` also formats `<script>` blocks with Prettier when `prettier` is on `PATH`; it isn't here or in CI,
  so formatting stays consistent. Installing Prettier globally would change `templ fmt`'s output.
- Packages: `internal/server` loads data and builds `views.Page`; `internal/views` renders it and holds date formatting;
  `internal/urls` builds every URL. Views never query the database.
- templ writes void elements without a trailing slash (`<meta ...>`); tests match attributes, not whole tags.
- Tests: logic is tested as plain functions (`newPage`, `newPagination`, `GroupByDay`, …) or by rendering one
  component in `internal/views`; HTTP tests check status codes, queries, and errors, not markup. templ's compiler
  catches missing fields and wrong arguments, so there are no tests that only check a field reaches the page.
- sqlc is pinned like templ: `go tool sqlc generate` after editing `internal/db/queries/*.sql`; CI runs
  `go tool sqlc diff`. Its first build compiles Postgres's C parser (about 25 s). `sqlc.yaml` reads the schema
  from Prisma's migration files, listed one by one. Overrides map `int4`/`serial` to `int` and
  `timestamptz` to `time.Time`, each twice (plain and nullable, which sqlc matches separately). Cast integer
  parameters and results to `::int`, not `::bigint`, or they need `int8` overrides too.
- Filters are optional parameters (`sqlc.narg(x) IS NULL OR ...`), so query text never changes. `Posts` and
  `CountPosts` repeat the same `WHERE`: keep them identical; `db_test.go` checks they agree.
- `db.Post` and the other row types are aliases of sqlc's generated structs (`PostsRow`, …). `PostFilter.Number` is a
  pointer because 0 is a valid post number (a zero value once made `/0` show the newest post).
- Database tests read only and run when `TEST_DATABASE_URL` is set, for example the production copy.
- `web/dist` is gitignored and embedded: run `npm run go:assets` before `go build`, `go test`, or `go vet`.
- `go.mod` has `ignore ./node_modules`: the `flatted` npm package ships a Go package that `./...` would otherwise pick up.
- Check gofmt with `gofmt -l cmd internal web`, not `.` (avoids `node_modules`).
- libvips comes from npm, not the system (see "govips and libvips"). Go commands that compile `internal/images/libvips`
  or `cmd/server` need `PKG_CONFIG_LIBDIR=$PWD/.libvips` (absolute: cgo runs `pkg-config` in each package's folder).
  `npm run go:dev` and `npm run go:test` set it; set it in the editor's Go settings so gopls can compile cgo.
  Other packages, including `internal/server`, compile without it.
- The file-writing tool can turn `\u` escapes in written Go source into literal characters; check non-ASCII strings.
- Local database: 31 posts, 18 published, all published ones cached at renderer version 1.
  Posts 1–5 share one `publishedAt` (seed data); queries order by `publishedAt DESC, number DESC`.
- Local Postgres runs with `TimeZone = Europe/Rome`.
- Running test servers next to the user's:
  - The user may have `go:dev` running on 6200: run test servers on 6201, built to the scratchpad and started
    directly, and stop them only by their own PIDs. Never `pkill -f` patterns that match the user's servers.
  - Start SvelteKit with `node node_modules/vite/bin/vite.js dev` so its PID is the server's, and use a
    `trap ... EXIT` so a failing command doesn't leave it running.
  - Add `options=-c%20TimeZone%3DUTC` to both apps' `DATABASE_URL`, or SvelteKit shows wrong times locally.
  - Quote URLs containing `?` in zsh, or it treats them as globs and the command fails.
  - Headless Chrome over the DevTools protocol (Node's built-in `WebSocket`, `--remote-debugging-port`) can
    compare `document.body.innerText`, emulate `prefers-color-scheme`, click controls, and take screenshots.

## Docker and deployment

- Stages: `assets` (Node, also writes `.libvips`) → `build` (Go, with `node_modules/@img` and `.libvips` at the same
  paths) → `runtime` (Debian trixie + sharp's `libvips-cpp.so`) → `server` (last, the default).
  The `assets` stage copies `internal/views` so Tailwind can scan the components; `build` uses the committed `_templ.go` files.
- `docker build --target libvips-test .` runs the image processing tests inside the runtime image.
- Image is about 146 MB (242 MB with Debian's libvips), runs as `nobody`; `.dockerignore` keeps `.env` and `node_modules` out.
- Railway probably builds a root `Dockerfile` automatically (unverified): don't merge `go-migration` into `main` before cutover.
- Production currently starts through `npm start`, whose launcher process used 55.5 MiB.

## govips and libvips

- libvips is sharp's prebuilt build from npm, 8.18.7 with mozjpeg: `@img/sharp-libvips-dev` (headers, dev dependency)
  and `@img/sharp-libvips-<os>-<arch>` (one library with every dependency built in; optional dependencies, so npm
  installs only the current platform's). `scripts/libvips.mjs` (`npm run go:libvips`) writes `.libvips/vips.pc`.
  The same version runs in development and production. Add a platform to `optionalDependencies` to build on it.
- `internal/images/libvips/magick.c` stubs `vips_magickload_buffer`: sharp's build has no ImageMagick, and govips calls
  it for formats like BMP, which now fail to load. The stub declares what it uses instead of including `vips.h`, or
  the linker warns about libvips's flags appearing twice.
- Comments directly above `import "C"` are compiled as C.
- The JPEG settings are sharp's with `mozjpeg: true`; outputs match sharp's within 1 KB.
- Install with `go get github.com/davidbyttow/govips/v2/vips@v2.20.0`. Asking for `.../govips/v2` resolves to a stale 2020 root-module version.
- `vips.Startup(&vips.Config{})` turns caching off: values ≥ 0 are applied. `Startup(nil)` uses govips defaults, with caching.
- Why not Debian's libvips: trixie ships 8.16.1 without mozjpeg, and files came out 14% (main) and 19% (preview)
  bigger. No Go binding bundles libvips, and building it from source in Docker costs minutes per uncached build.

## Uploads

- `POST /{number}/edit/upload`, multipart field `file`, up to 20 MB. Answers `{main, preview, width, height}` like
  SvelteKit; a file libvips can't read is a 400, and without the `STORAGE_S3_*` variables uploads answer 503.
- `STORAGE_S3_*` are all set or all unset; the local `.env` has none, so uploads are off locally by default.
- `internal/images` is plain Go (keys, the two `Put`s, interfaces), so the server and its tests don't need libvips.
- Non-ASCII file names go into the `original` metadata as RFC 2047 (`=?utf-8?q?…?=`): S3 metadata must be ASCII.
  SvelteKit's JS SDK would have failed on them.
- The SDK addresses the bucket in the host name (`bucket.endpoint`), like SvelteKit's.
- Local S3 for testing: `rclone serve s3 :memory: --addr 127.0.0.1:9100 --auth-key test,test --force-path-style=false`
  (rclone is installed; nothing is stored on disk). Create the bucket with a signed `PUT http://uploads.localhost:9100/`
  (`curl --aws-sigv4 'aws:amz:us-east-1:s3' --user test:test`), then run the server with `STORAGE_S3_ENDPOINT=http://localhost:9100`.
  MinIO no longer publishes images on Docker Hub or Quay.
- Tested end to end (2026-11-21): a blog photo gave a 174 KB main image and a 99 KB 1200×1073 preview (sharp: 175 and
  100 KB), with `image/jpeg` and the encoded file name.

## SvelteKit behavior

- Production `BODY_SIZE_LIMIT` is `20M`: the Go upload limit must match.
- Uploads: keys `YYYY/MM/<12 hex>.jpg` and `…_preview.jpg` (server-local month); main JPEG q85, preview 1200 px
  wide q80, both mozjpeg with auto-orientation; the response carries the preview's dimensions; S3 metadata `original`.
- Sessions: cookie `notedimatteo_session`, 60 days, `httpOnly`, `sameSite=lax`, `secure` outside dev;
  `lastActivityAt` updated at most once a minute and has no database default (Prisma `@updatedAt`), so inserts set it.
- Prisma 7's `@prisma/adapter-pg` replaces the offset of every `timestamptz` it reads with `+00:00`, so times are
  only right when the session time zone is UTC. Go reads them correctly.
- Archive months use `EXTRACT` in the database session's time zone, not Rome's; Go keeps the same SQL (TODO).
- Tags inside a post have no order column and no `ORDER BY`, in both apps: whatever order Postgres returns.
  Locally that's the order saved from the editor; on the production copy, Postgres reads them through the
  `(postId, tagName)` index, so they come sorted by name. Both apps showed the same order. Kept as is on purpose.
- Theme: mode-watcher stores the preference in localStorage `mode-watcher-mode` (`system`/`light`/`dark`); dark mode
  is a `.dark` class. Go applies it with an inline head script (`themeInit` in `layout.templ`) and handles the
  select in `web/assets/site.ts`.
- GoatCounter is disabled when the host isn't `notedimatteo.it` (Go uses the host of `SITE_URL`).

## Markdown renderer (`internal/markdown`)

- goldmark with strikethrough (`~` and `~~`), raw HTML dropped (inline tags removed, text kept), and `[img]`
  reimplemented after the rehype plugin: the `_preview` variant, linked to the original or to `link`, lazy-loaded.
- remark merges adjacent text where goldmark splits it (around brackets that aren't links), so the transformer
  matches `[img]` across runs of adjacent text nodes; a hard line break ends a run.
- Fixed on purpose, with no effect on the 670 production posts (`image_test.go`): `[img]` in code stays as written;
  `_preview` goes before the path's extension; an empty path doesn't match; `link` must be http(s); `width` and
  `height` must be integers.
- `markdown.Version` is 2 (SvelteKit's is 1) and is stored in `post_html_cache`. Bump it whenever the output bytes
  change. Same structure as SvelteKit's but not the same bytes: Go ends with a newline and escapes `"` and `&` as
  `&quot;`/`&amp;` where SvelteKit writes `"`/`&#x26;`. The cutover and a rollback re-render each post once.
- Cache: posts without an up-to-date entry are rendered and saved with an `ON CONFLICT` upsert. Unlike SvelteKit, a
  failed cache write is logged and the page is still shown.
- `STORAGE_PUBLIC_BASE_URL` is required by the Go server.
- Compatibility test: `testdata/compat/*.md` with the HTML the SvelteKit renderer produces, compared by parsed
  structure (decoded text, sorted attributes, CRLF normalized). Regenerate expected files with
  `node internal/markdown/testdata/generate.mjs`, which imports the app's own plugins; it stops working once
  SvelteKit is removed, and the fixtures stay as they are.
- Against production: export each post's body as `.md` and its cached HTML as `.html` (that cache is SvelteKit's
  actual output), then run `MARKDOWN_COMPAT_DIR=<dir> MARKDOWN_COMPAT_STORAGE_URL=https://uploads.notedimatteo.it
  go test ./internal/markdown`. All 670 posts match (backup of 2026-11-20).

## Authentication and drafts

- `ADMIN_USERNAME` and `ADMIN_PASSWORD` are required, as in SvelteKit. Go compares them in constant time.
- `handle` (`internal/server/middleware.go`) loads the session before every page handler. Assets skip it, and a page
  request queries `sessions` only when it carries the cookie.
- New tokens come from `crypto/rand.Text()` (26 characters). SvelteKit's 64-character hex tokens keep working,
  because the server only looks tokens up.
- The cookie isn't `Secure` when the request's host is localhost or a loopback address. `SITE_URL` can't decide
  this: local `.env` keeps it at notedimatteo.it, or GoatCounter would count local visits.
- Changed on purpose: logout and "new" are POST forms (`/logout`, `/unpublished/new`), because Lax cookies reach
  GET links from other sites. Their old GET URLs now return 404. The delete form posts to `/{number}/delete`,
  which deletes only drafts; a published post gets a 404.
- `http.CrossOriginProtection` returns the 403 error page for POSTs from other sites.
- The "modifica" links lead to `/{number}/edit`, which Go serves only from step 5 (404 until then).
- TODOs: expire sessions in the database, and limit failed logins. SvelteKit does neither.
- Tested by hand against the local database: login, the drafts list (same text as SvelteKit), creating and
  deleting a draft, a cross-site POST, and logout.

## Editor endpoints

- `POST /{number}/edit` saves the form (multipart `FormData` or URL-encoded). Fields as in SvelteKit: `intent`
  (`save`, `publish`, `unpublish`), `number`, `category`, `title`, `slug`, `hideTitle` (`true` or `on`), `body`,
  `tags` (comma-separated). It answers `{"post": {...}}`, or 400 with `{"errors": {"field": "message"}}`.
- `POST /{number}/edit/preview` takes `{"body": "..."}` and answers `{"html": "..."}`, like SvelteKit.
- Both use `handleAPI`: errors are JSON (`{"error": "Not Found"}`), and logged-out requests get 401, not a redirect.
  Bodies over 1 MB get 413.
- Same as SvelteKit: publishing sets `publishedAt` to now even if the post was published; every save sets
  `updatedAt`; the body revision goes up only when the body changes; tags without posts stay in `tags`.
- Changed on purpose: an unknown `intent` is a 400 (SvelteKit saved without changing publication); a number
  that another post has is a field error (SvelteKit: 500).
- `db.SavePost` locks the post's row (`FOR UPDATE`), then updates the post, its cached HTML, and its tags in
  one transaction. The server renders the HTML before the transaction starts.
- `pgx` returns times in the Go process's time zone, so the JSON has `+01:00` offsets locally. The instants are right.
- Database tests that write use `openTestTx`, a transaction rolled back at the end, so they're safe on the
  production copy.
- Pitfall: `ParseMultipartForm` hides `ParseForm`'s errors for URL-encoded bodies, so `parseForm` calls
  `ParseForm` first.

## Public pages (`scripts/parity.py`)

- Compares SvelteKit (6100) and Go (6201) for a list of paths: status, `Location`, title, meta, canonical,
  JSON-LD, link `href`s (resolved and decoded), visible text, and the feed as parsed XML.
  `python3 -I scripts/parity.py http://localhost:6100 http://localhost:6201 <paths file> <report file>`.
- Run both apps on `notedimatteo_prod` with `options=-c%20TimeZone%3DUTC` and
  `STORAGE_PUBLIC_BASE_URL=https://uploads.notedimatteo.it` for both, or SvelteKit caches HTML with the dev R2 URL.
- Paths used (905): every post's canonical URL, stale slugs, every tag, category, and month, home pages, search
  terms, errors, redirects, and unusual paths. Normalized on purpose: SvelteKit's relative links and redirect
  locations, sidebar tags with equal counts (ordered by name in Go), feed `content:encoded` as decoded HTML.
- Intended differences (40 paths): `?page=` that isn't a positive integer is a 400 (SvelteKit: 500, or `2abc` as
  page 2); paths with `//`, `.`, or `..` get the router's 307 to the clean path (SvelteKit: 404 or 200); every 404
  says "Not Found" (SvelteKit: "Error: 404" for some); search escapes `%` and `_`; descriptions are cut by
  characters, so #651's ends one character later (an emoji at the limit).
- Known difference: SvelteKit's server render never marks the current category chip, because its relative link
  isn't equal to the path; hydration fixes it. Go marks it in the HTML, like the hydrated page.
- Routes: Go's `ServeMux` panics on overlapping patterns, so assets use `/assets/{file}` (Vite's output is flat)
  rather than `/assets/`, which would overlap `/{number}/{slug}`.
- Errors show the status and its standard text inside the layout, without the sidebar. Redirects have no body.
  A trailing slash is a 308 to the path without it.
- Only pages with a sidebar load the archive and tag counts (`newSidebarPage`).
- JSON-LD is encoded with `json.Marshal`, which escapes `<`, `>`, and `&` (same JSON, safe inside `<script>`).

## Production data

- Backups: private repo `matteocontrini/notedimatteo-backups`, committed daily by a GitHub Action running
  `supabase db dump`: `schema.sql`, `data.sql` (`COPY`), and an almost empty `roles.sql`.
- Import: create an empty database, then `psql "<url>" -v ON_ERROR_STOP=1 -f schema.sql -f data.sql`.
  The 2026-11-20 backup is imported locally as `notedimatteo_prod`; the original `notedimatteo` is unchanged.
- To use it, point `DATABASE_URL` in `.env` at `notedimatteo_prod`: `go:dev` sources `.env`, which overrides
  a `DATABASE_URL` set on the command line.
- 2026-11-20: 670 posts (#1–#670), all published, no drafts; 99 tags; 1,095 post–tag links; all cached at
  renderer version 1. 189 posts use `[img]`, 43 contain backticks, 10 contain `<`, none have CRLF line endings,
  none contain `]]>`, and no two share a `publishedAt`.
- Images are served from `https://uploads.notedimatteo.it`.
- Production Postgres runs with `TimeZone = Etc/UTC`, so the Prisma 7 time zone bug doesn't affect the live site,
  and archive months are grouped in UTC. Local comparisons force a UTC session to match.
- After a comparison run, restore the production copy's cache, which both apps overwrite: extract the
  `post_html_cache` `COPY` block from `data.sql`, then `TRUNCATE post_html_cache` and load it in one transaction.
- `sessions` holds 11 production login tokens: treat the local copy like the backup repo.

## Blog (`../blog`)

- Hugo 0.165.0, PaperMod theme (git submodule), Netlify. `defaultContentLanguage: en`.
- URLs: English posts at `/:slug/`, Italian at `/it/:slug/`; lists at `/blog/` and `/it/blog/`.
- `_redirects`: `blog.contrini.it/*` → `matteosonoio.it/:splat`. Netlify headers: `nosniff`, `X-Frame-Options: DENY`,
  CSP `frame-ancestors 'none'`.
- Content: 32 Markdown posts (12 English, 20 Italian) as page bundles, 77 MB of media
  (67 jpg, 85 png, 5 webp, 1 svg, 9 mp4).
- Front matter: `title`, `date`, `lastmod`, `slug`, `summary`, `showtoc` (19 posts), occasionally `cover`, `layout`, `url`.
- Shortcodes: `fig` 144 times (`src`, `vertical`, `nolink`; serves WebP when smaller, links the original),
  `relref` 6, `center` 1. Partials: giscus comments, a "written by a human" image after every post, a DMARCwise plug.
- goldmark with `unsafe: true`: raw `<sup>` 19 times, `<video>` 6 times.
- About 90 fenced code blocks with languages (csharp 15, html 14, sh 13, c 9, json 8, …); tables in 3 posts, footnotes in 3.
- giscus: repo `matteocontrini/blog`, category `Comments`, `data-mapping="pathname"`: changing an article's path detaches its thread.
- `scripts/prepare_images.py` converts ICC-tagged images to sRGB.

## Design canvas

- https://claude.ai/artifact/DeA9bBLoajjHfTZfgk1uU2. Picked boards: E2/E4 home, D2 notes, D3 article,
  D4 article archive, F2 projects, G1–G3 editor. A sticky note on the canvas describes the CodeMirror approach.
- Fonts: Public Sans (interface) and Source Serif 4 (prose). Colors: ink `#1D1D20`, muted `#5E6066`, link `#1F3FBF`.
- Category colors: note `#1F3FBF`, cit `#A0189E`, segnalibri `#00838A`, til `#9A7B00`, video `#C0281E`, progetti `#1E8A2E`.
- Notes stay Italian-only; articles have a language; the article page shows related notes by tag and giscus comments.
