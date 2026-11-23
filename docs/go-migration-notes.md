# Go migration notes

Working notes for `go-migration-plan.md`: facts found while working and gotchas. The
migration is done; `ARCHITECTURE.md` describes the result, and `redesign-notes.md` continues. Record what the code and git history
don't show; remove notes that stop being true.

## Working agreements

- Parity means the same pages, URLs, and behavior for what visitors and existing links produce. Edge-case
  input handling (number parsing, odd paths, error wording) uses plain, idiomatic Go instead of emulating SvelteKit.
  Visible fixes that change real content are deferred as `TODO` comments (`grep -rn TODO cmd internal web`).
- The user handles git: never commit, push, or stage.
- Keep plan and notes succinct and schematic.

## Local development

- `npm run dev`: port 6200 (runs `generate`, sources `.env`, `go run`).
- Scripts by purpose (2026-11-21): `generate` (templ, sqlc, Vite: about 6 s), `format`, `check` (`check:js`, `check:go`),
  `test` (`test:js`, `test:go`, `test:e2e`). `dev`, `check`, `test:go`, and `test:e2e` run `generate` first, so nobody
  has to remember it. CI's `check` job fails when `generate` leaves changes.
- No hot reload: restart `npm run dev` after template or CSS changes.
- templ is pinned with the `tool` directive in `go.mod`: run it as `go tool templ` (`generate`, `fmt`, `lsp` for editors).
  Generated `*_templ.go` files are committed; after editing a `.templ` file, run `go tool templ generate` and `templ fmt`.
- `templ fmt` formats `<script>` blocks with Prettier when `prettier` is on `PATH`. npm scripts have
  `node_modules/.bin` on `PATH`, so `format` and `check` pass `-prettier-required`: a direct `go tool templ fmt`
  without Prettier would format differently.
- Packages: `internal/server` loads data and builds `views.Page`; `internal/views` renders it and holds date formatting;
  `internal/urls` builds every URL. Views never query the database.
- templ writes void elements without a trailing slash (`<meta ...>`); tests match attributes, not whole tags.
- Tests: logic is tested as plain functions (`newPage`, `newPagination`, `GroupByDay`, …) or by rendering one
  component in `internal/views`; HTTP tests check status codes, queries, and errors, not markup. templ's compiler
  catches missing fields and wrong arguments, so there are no tests that only check a field reaches the page.
- sqlc is pinned like templ: `go tool sqlc generate` after editing `internal/db/queries/*.sql`; CI runs
  `go tool sqlc diff`. Its first build compiles Postgres's C parser (about 25 s). `sqlc.yaml` reads the schema
  from the goose migrations in `internal/db/migrations/`. Overrides map `int4`/`serial` to `int` and
  `timestamptz` to `time.Time`, each twice (plain and nullable, which sqlc matches separately). Cast integer
  parameters and results to `::int`, not `::bigint`, or they need `int8` overrides too.
- Filters are optional parameters (`sqlc.narg(x) IS NULL OR ...`), so query text never changes. `Posts` and
  `CountPosts` repeat the same `WHERE`: keep them identical; `db_test.go` checks they agree.
- `db.Post` and the other row types are aliases of sqlc's generated structs (`PostsRow`, …). `PostFilter.Number` is a
  pointer because 0 is a valid post number (a zero value once made `/0` show the newest post).
- Database tests: `TestMain` in `internal/db` starts `postgres:18` with testcontainers-go, applies the migrations, and
  loads `testSeed` (4 published posts, 1 draft, 2 tags, 2 months). `-short` skips them. `TEST_DATABASE_URL` uses
  another database as it is, without migrations or seed: the production copy works, and writes roll back.
  testcontainers-go added many modules to `go.mod` (Docker client, OpenTelemetry); only test binaries compile them.
- `web/assets/main.css` holds what was `src/app.css`.
- `web/dist` is gitignored and embedded: run `npm run generate` before a bare `go build`, `go test`, or `go vet`.
- `go.mod` has `ignore ./node_modules`: the `flatted` npm package ships a Go package that `./...` would otherwise pick up.
- Check gofmt with `gofmt -l cmd internal web`, not `.` (avoids `node_modules`).
- libvips comes from npm, not the system (see "govips and libvips"). Go commands that compile `internal/images/libvips`
  or `cmd/server` need `PKG_CONFIG_LIBDIR=$PWD/.libvips` (absolute: cgo runs `pkg-config` in each package's folder).
  The npm scripts that run `go` set it; set it in the editor's Go settings so gopls can compile cgo.
  Other packages, including `internal/server`, compile without it.
- cgo still needs the `pkg-config` command (Homebrew `pkgconf` on the Mac; the Dockerfile installs it).
- End-to-end checks: build the server to the scratchpad, run it on 6201 against the local `notedimatteo` database,
  and drive it with curl (log in, act, log out so no session rows are left). Upload checks use rclone's S3 (see "Uploads").
- The file-writing tool can turn `\u` escapes in written Go source into literal characters; check non-ASCII strings.
- Local database (2026-11-21): 30 posts (numbers up to 33), 18 published. Draft #29 disappeared on 2026-11-21,
  probably deleted from the browser; tests don't delete posts. Test scripts delete the drafts they create.
  Posts 1–5 share one `publishedAt` (seed data); queries order by `publishedAt DESC, number DESC`.
- Local Postgres runs with `TimeZone = Europe/Rome`.
- Running test servers next to the user's:
  - The user may have `npm run dev` running on 6200: run test servers on 6201, built to the scratchpad and started
    directly, and stop them only by their own PIDs. Never `pkill -f` patterns that match the user's servers.
  - Quote URLs containing `?` in zsh, or it treats them as globs and the command fails.
  - A headless Chrome left running keeps its DevTools port: check the port is free, and stop leftovers by PID.

## Docker and deployment

- CI: the `docker` job runs both `docker build`s in parallel with `check` and `test`. `test` is a matrix that runs
  `test:js`, `test:go`, and `test:e2e` on three runners: in one job they took 110 s in a row, a minute more than in parallel. A GitHub layer cache (`type=gha`,
  `mode=max`) was tried and removed: its first export took 188 s, more than the installs it saves (`npm ci` 39 s,
  `go mod download` 13 s), and every lockfile change would upload the big layers again.
- Stages: `assets` (Node, also writes `.libvips`) → `build` (Go, with `node_modules/@img` and `.libvips` at the same
  paths) → `runtime` (Debian trixie + sharp's `libvips-cpp.so`) → `server` (last, the default).
  The `assets` stage copies `internal/views` so Tailwind can scan the components; `build` uses the committed `_templ.go` files.
- `docker build --target libvips-test .` runs the image processing tests inside the runtime image.
- Image is about 146 MB (242 MB with Debian's libvips), runs as `nobody`; `.dockerignore` keeps `.env` and `node_modules` out.
- Railway probably builds a root `Dockerfile` automatically (unverified): don't merge `go-migration` into `main` before cutover.
- Memory on Railway: SvelteKit used 500 MB or more (its `npm start` launcher alone used 55.5 MiB). After cutover
  (2026-11-21), the Go server uses about 16 MB.

## Cutover and after

- Order (decided 2026-11-21): remove SvelteKit and update CI → goose → browser tests → deploy → watch →
  drop `_prisma_migrations` → Renovate. There's no second Railway service. The first Go deploy has the browser
  tests behind it.
- Goose: the baseline is the current schema (`internal/db/schema/`), so production's schema doesn't change.
  Keep `_prisma_migrations` until a rollback is no longer needed: if anything ran `prisma migrate deploy` on
  a rolled-back SvelteKit, Prisma would try to apply its migrations again. goose didn't run at server start
  before cutover, so the cutover changed nothing in the database.
- Browser tests follow `../dmw/certs-email`: Playwright, a global setup that starts the app, testcontainers,
  a few long flows with `test.step`, and the report uploaded on failure.
- rclone's S3 runs from the `rclone/rclone` image through testcontainers, so the tests start it the same way
  locally and in CI, and nobody needs rclone installed. A GitHub Actions service would also work: services take
  `command` and `entrypoint`. Distro packages are too old for `serve s3` (1.65+).
- The SDK reaches the bucket at `uploads.localhost:<port>`: macOS and the GitHub Actions runner both resolve it.
- Renovate (`renovate.json`, 2026-11-21), from certs-email's config: server packages (Go modules, Go version, libvips)
  weekly on Wednesdays; build and browser tooling monthly on the 15th (the 1st is a busy day); no vulnerability
  PRs, as in certs-email. gomod with
  `go mod tidy`; the `go` directive and the `golang` image in one group; templ and sqlc in their own group,
  because their PRs need `npm run generate` pushed by hand (CI checks the generated code); sqlc is `// indirect`
  in `go.mod`, which Renovate skips unless a rule enables it; `@img/sharp-libvips-*` in one group; rclone's image
  through a regex manager, `postgres:18` by hand (production's major). Validated with `renovate-config-validator`.
- Lockfile maintenance: npm enforces the age (`min-release-age=3` in `.npmrc`, like dmw-frontend). Renovate can't
  date the packages a refresh changes, so its `renovate/stability-days` check stays pending, unless
  `minimumReleaseAgePackageManagerSync` (experimental) is on, `.npmrc`'s age is at least `minimumReleaseAge`, and
  the npm constraint is within `>=11.10.0` (hence `^11.10.0`). All three are set.
- GitHub Actions use exact versions, locked in `.github/workflows/actions.lock` by `gh actions-lock` (v0.4.1,
  like certs-email); the `verify-lockfile` job checks it. After editing a workflow's `uses:`, run `gh actions-lock`.

## Goose

- `internal/db/migrations/00001_baseline.sql` is the schema Prisma's three migrations created, with `IF NOT EXISTS`
  everywhere and the foreign keys inside `CREATE TABLE` (Postgres has no `ADD CONSTRAINT IF NOT EXISTS`). On an
  existing database it changes nothing, and goose records version 1. It has no Down.
- `db.Migrate` runs goose's library (`goose.NewProvider`) on the embedded files, through pgx's `database/sql`
  driver; goose's CLI would pull in MySQL, SQLite, and ClickHouse drivers. The server runs it at
  startup, before opening its pool (since 2026-11-21, after cutover); `cmd/migrate` and `npm run migrate` are gone.
  No session lock: Railway runs one instance, and the old one never migrates.
- Checked 2026-11-21 with `pg_dump --schema-only` on two scratch databases: an empty one after the migration, and a
  copy of `notedimatteo_prod` after it, both match the production schema exactly (column order, defaults,
  sequences, constraints). The copy kept its data and `_prisma_migrations`. A second run applies nothing.
- goose ignores files without a numeric prefix silently; `TestMigrationsLoad` catches duplicate versions.
- Adding goose raised pgx to 5.12.0 (then upgraded to 5.13.0, the latest) and some of sqlc's and templ's modules (grpc, protobuf, x/tools),
  through goose's own requirements.
- The first start of the server on a database (production, local ones) adds `goose_db_version` and records version 1.
- `00002_drop_prisma_migrations.sql` drops `_prisma_migrations` (2026-11-21). Checked on a copy of `notedimatteo_prod`:
  both migrations apply, the 670 posts stay. A SvelteKit rollback still runs (`npm start` never ran Prisma's
  migrations), but would need the table back before any `prisma migrate deploy`.

## Browser tests (`tests/e2e`)

- `npm run test:e2e` runs `generate`, builds `.e2e/server`, then runs Playwright. The global setup starts
  `postgres:18` and `rclone/rclone:1.74.4` with testcontainers, starts the server on port 6210 (which applies the
  migrations), then loads the seed from `data.ts` with `psql` inside the Postgres container.
- rclone serves one directory twice: S3 on 9000 (the bucket is a subdirectory, so no bucket creation) and plain
  HTTP on 8080 for the bucket, which is `STORAGE_PUBLIC_BASE_URL`, so uploaded images load in the browser.
- `SITE_URL` is `https://notedimatteo.it`, so the page turns GoatCounter off; the fixture also aborts requests to
  other hosts. The fixture fails a test on any uncaught JavaScript error (checked by breaking the editor's script).
- Flows: reading (pagination, slug redirect, category, tag, month, search, feed, 404, theme), login and logout,
  and writing a note (field errors, shortcuts, preview, tag Tab completion, pasted and chosen uploads, publish,
  the note on the home page with its image and in the feed, a taken number, unpublish, delete). About 10 s locally.
- The editor's "Preview" and "Back to editing" buttons are inside the Body `<label>`, so their accessible name
  is "Body"; the test matches them by text. Markup kept as it was.
- Postgres 18 because local is 18.6; production's version isn't in the backup dump.
- Pass on GitHub Actions (2026-11-21), so Linux resolves `uploads.localhost` too.

## govips and libvips

- libvips is sharp's prebuilt build from npm, 8.18.7 with mozjpeg: `@img/sharp-libvips-dev` (headers, dev dependency)
  and `@img/sharp-libvips-<os>-<arch>` (one library with every dependency built in; optional dependencies, so npm
  installs only the current platform's). `scripts/libvips.mjs` (`postinstall`) writes `.libvips/vips.pc`.
  The same version runs in development and production. Add a platform to `optionalDependencies` to build on it.
- `internal/images/libvips/magick.c` stubs `vips_magickload_buffer`: sharp's build has no ImageMagick, and govips calls
  it for formats like BMP, which now fail to load. The stub declares what it uses instead of including `vips.h`, or
  the linker warns about libvips's flags appearing twice.
- Comments directly above `import "C"` are compiled as C.
- The JPEG settings are sharp's with `mozjpeg: true`; outputs match sharp's within 1 KB.
- Like sharp, `Process` converts images with an embedded ICC profile to sRGB, then strips the profile. Dropping the
  profile without converting made the blog's WebPs look washed out (Hugo, September 2026). A test converts an sRGB
  color to Display P3 with libvips's built-in `"p3"` profile and checks that the output has the sRGB values again.
- Install with `go get github.com/davidbyttow/govips/v2/vips@v2.20.0`. Asking for `.../govips/v2` resolves to a stale 2020 root-module version.
- `vips.Startup` gets cache limits of 0, which turn caching off: govips applies values ≥ 0. `Startup(nil)` uses govips
  defaults, with a cache of 50 MB and 100 operations.
- Memory (2026-11-22): production (1 GB, 1 vCPU) was killed without a log line when uploading two 2900×2400 PNG
  screenshots. The first upload took it from 21 MB to 917 MB, where it stayed. `ConcurrencyLevel: 0` meant a libvips
  thread per host CPU, each with its own buffers, and glibc's allocator kept the freed memory in a pool per thread.
  Docker repro, 8 rounds of 2 uploads at once of the same screenshot (RSS after):
  - 64 threads: 1.25 GB after round 1, 2 GB after round 4
  - 64 threads, `MALLOC_ARENA_MAX=2`: about 1.1 GB, flat
  - 1 thread: grows by about 100 MB a round
  - 1 thread, `MALLOC_ARENA_MAX=2`: about 220 MB, flat; same speed (about 650 ms an image)
  - plus `debug.FreeOSMemory()` after each round: no change (Go's own memory is small)
  - plus glibc's `malloc_trim(0)` after each round (chosen, after each upload): back to about 25 MB
  - plus `malloc_trim(0)` but without `MALLOC_ARENA_MAX`: 160 MB after 12 rounds
  - imgproxy does the same: one libvips thread, `MALLOC_ARENA_MAX=2`, and `FreeOSMemory` plus `malloc_trim(0)` every
    10 s. sharp also runs libvips with one thread on glibc. jemalloc would replace `MALLOC_ARENA_MAX` but adds a package.
- Rejected (2026-11-21): Homebrew's `vips` (8.18.7 with mozjpeg, but 28 dependencies, ImageMagick and Poppler
  included), and development inside Docker (the image would need sharp's libvips anyway, and testcontainers,
  Playwright's server, and GoLand's cgo insight would all get harder).
- Why not Debian's libvips: trixie ships 8.16.1 without mozjpeg, and files came out 14% (main) and 19% (preview)
  bigger. No Go binding bundles libvips, and building it from source in Docker costs minutes per uncached build.

## Uploads

- `POST /{number}/edit/upload`, multipart field `file`, up to 20 MB. Answers `{main, preview, width, height}` like
  SvelteKit; a file libvips can't read is a 400, and without the `STORAGE_S3_*` variables uploads answer 503.
  Other processing errors are 500s.
- An upload continues if the client goes away (`context.WithoutCancel`), so the main image is never stored without
  its preview. SvelteKit's did the same. The server's 30 s timeouts apply to uploads too: decided that's fine.
- `STORAGE_S3_*` are all set or all unset. The local `.env` points them at the dev R2 bucket
  (`dev-notedimatteo-uploads`): test servers override them with rclone's.
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

- Production `BODY_SIZE_LIMIT` was `20M`, so the Go upload limit is 20 MB.
- Uploads: keys `YYYY/MM/<12 hex>.jpg` and `…_preview.jpg` (server-local month); main JPEG q85, preview 1200 px
  wide q80, both mozjpeg with auto-orientation; the response carries the preview's dimensions; S3 metadata `original`.
- Sessions: cookie `notedimatteo_session`, 60 days, `httpOnly`, `sameSite=lax`, `secure` outside dev;
  `lastActivityAt` updated at most once a minute and has no database default (Prisma `@updatedAt`), so inserts set it.
- Prisma 7's `@prisma/adapter-pg` replaces the offset of every `timestamptz` it reads with `+00:00`, so times are
  only right when the session time zone is UTC. Go reads them correctly.
- Archive months use `EXTRACT` in the database session's time zone, not Rome's; Go keeps the same SQL (TODO).
  Since 2026-11-23, Go groups them in Europe/Rome, and the month pages select in Rome too (`sqlite-notes.md`).
- Tags inside a post have no order column and no `ORDER BY`, in both apps: whatever order Postgres returns.
  Locally that's the order saved from the editor; on the production copy, Postgres reads them through the
  `(postId, tagName)` index, so they come sorted by name. Both apps showed the same order. Kept as is on purpose.
  Since `00004_tag_position.sql` (2026-11-23), a `position` column keeps the order typed (`sqlite-notes.md`).
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
  `_preview` goes before the path's extension; an empty path doesn't match; `link` must be http(s) or a path on
  the site like `/670`; `width` and `height` must be integers; `[img]` inside a normal image's alt text stays as
  written.
- Accepted differences for future posts (none of the 670 has them):
  - goldmark's strikethrough matches tilde runs of different lengths: `~a~~` is `<del>a</del>~` (remark: literal).
  - Without raw HTML, goldmark empties `javascript:`, `vbscript:`, `file:`, and non-image `data:` URLs in links
    and images.
  - An `[img]` path can't contain brackets: `[img]a[1].jpg[/img]` stays text.
- `markdown.Version` is 2 (SvelteKit's is 1) and is stored in `post_html_cache`. Bump it whenever the output bytes
  change. Same structure as SvelteKit's but not the same bytes: Go ends with a newline and escapes `"` and `&` as
  `&quot;`/`&amp;` where SvelteKit writes `"`/`&#x26;`; text in a run with `[img]` uses `&#34;` and `&#39;`. The cutover and a rollback re-render each post once.
- Cache: posts without an up-to-date entry are rendered and saved with an `ON CONFLICT` upsert. Unlike SvelteKit, a
  failed cache write is logged and the page is still shown.
- `STORAGE_PUBLIC_BASE_URL` is required by the Go server.
- Golden test: `testdata/golden/*.md` with Go's exact output in `.html`, compared byte for byte;
  `go test ./internal/markdown -update` rewrites them after an intended change. Until 2026-11-21 the same inputs
  were compared by structure with SvelteKit's output; the switch only changed escaping and the final newline.
- Against production (2026-11-20, before the golden switch): all 670 posts matched SvelteKit's cached HTML by
  structure. That comparison mode and its `golang.org/x/net/html` normalizer were removed with the golden switch.

## Authentication and drafts

- `ADMIN_USERNAME` and `ADMIN_PASSWORD` are required, as in SvelteKit. Go compares them in constant time.
- `handle` (`internal/server/middleware.go`) loads the session before every page handler. Assets skip it, and a page
  request queries `sessions` only when it carries the cookie.
- New tokens come from `crypto/rand.Text()` (26 characters). SvelteKit's 64-character hex tokens keep working,
  because the server only looks tokens up.
- The cookie isn't `Secure` when the request's host is localhost or a loopback address. `SITE_URL` can't decide
  this: local `.env` keeps it at notedimatteo.it, or GoatCounter would count local visits. Logging in to the dev
  server from another device (`http://192.168…`) fails, because the cookie is `Secure` there.
- Accepted: the cross-site 403 page shows the logged-out top bar (it skips the session). A logged-out
  `/abc/edit` redirects to the login page instead of returning 404.
- Changed on purpose: logout and "new" are POST forms (`/logout`, `/unpublished/new`), because Lax cookies reach
  GET links from other sites. Their old GET URLs now return 404. The delete form posts to `/{number}/delete`,
  which deletes only drafts; a published post gets a 404.
- `http.CrossOriginProtection` returns the 403 error page for POSTs from other sites.
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

## Editor (`web/editor`)

- `GET /{number}/edit` (login required) renders the layout, the editor's data in `<script id="editor-data"
  type="application/json">` (`templ.JSONScript`, which escapes `<`), and the `editor/main.ts` bundle.
  `postWithSlug` serves it: a `GET /{number}/edit` pattern would conflict with `GET /archivio/{tag}`.
- The editor started as a copy of SvelteKit's components and helpers; SvelteKit's copy was deleted in step 6.
- Each post in the JSON carries its URLs (`post`, `edit`, `preview`, `upload`), built by `internal/urls`.
  After a save that changes the number, the editor uses the new ones and `history.replaceState`s the address.
  SvelteKit showed a 404 there, because it reloaded the old number.
- `form.ts` holds the requests and dirty tracking: the form is dirty when it differs from what the last
  successful save sent, so typing during a save stays unsaved. Errors other than field errors show an `alert`,
  like Superforms' `onError`. ⌘S always saves, even when the Save button is disabled, like SvelteKit's.
- Superforms had no `taintedMessage`, so leaving with unsaved changes doesn't warn, then or now.
- Accepted differences from Superforms: a new draft doesn't show "Body is required" before its first save; fields
  keep what the user typed instead of the server's trimmed values; tag suggestions refresh only on reload; ⌘S
  skips the browser's own field validation, so the server's message shows instead.
- Known difference: the top bar's unpublished count doesn't change after publishing until the next page load
  (SvelteKit reloaded the layout's data after every save). Left as is: the redesign may drop the count.
- `assetResolver` refuses manifest entries with `css` or `imports`: pages load each entry with one tag.
  The editor's components have no `<style>`, so the bundle is one 84 KB file (31 KB gzipped).
- `npm run test:js` (Vitest) runs the editor's tests.
- A headless Chrome script (since deleted; Playwright tests will replace it) checked login, the drafts page, and the
  editor.
- 2026-11-21: all 30 checks pass on the local database and on the production copy. Real posts (#1, #670, and
  #152 with 4 images) load in the editor with the same body and tags as the database. The run re-rendered
  20 cached posts on the production copy; restoring the cache fixed them.

## Public pages

- `scripts/parity.py` (deleted in step 6) compared SvelteKit and Go on the production copy for a list of paths:
  status, `Location`, title, meta, canonical, JSON-LD, link `href`s (resolved and decoded), visible text, and the
  feed as parsed XML.
- Paths used (905): every post's canonical URL, stale slugs, every tag, category, and month, home pages, search
  terms, errors, redirects, and unusual paths. Normalized on purpose: SvelteKit's relative links and redirect
  locations, sidebar tags with equal counts (ordered by name in Go), feed `content:encoded` as decoded HTML.
- Intended differences (40 paths): `?page=` that isn't a positive integer is a 400 (SvelteKit: 500, or `2abc` as
  page 2); paths with `//`, `.`, or `..` get the router's 307 to the clean path (SvelteKit: 404 or 200); every 404
  says "Not Found" (SvelteKit: "Error: 404" for some); search escapes `%` and `_`; descriptions (meta and
  feed) are cut by characters, so #651's ends one character later (an emoji at the limit).
- Canonical and pagination links keep the request path's encoding (`EscapedPath`), like SvelteKit's.
  `parity.py` decodes URLs, so it can't see this. Post numbers beyond int32 are 404s.
- GoatCounter will count more page views after cutover: every Go page is a full load, while SvelteKit counted only
  a visit's first page.
- Go doesn't compress responses. Railway's edge gzips responses with a `Content-Length` (seen after cutover: CSS
  yes; HTML and the feed, streamed without one, no). Pages and the feed therefore set `Content-Length`
  from their buffers, and the edge now gzips them (checked 2026-11-22). Pages get `no-cache`, and
  `private, no-cache` for a logged-in user (admin controls); SvelteKit sent no `Cache-Control` on pages.
- No `ETag`s, though SvelteKit sent them on pages: Go has no built-in support for dynamic pages, and a 304
  still costs the queries and the render. Hashed assets are `immutable`; the favicon is refetched daily (6.8 KB).
- SvelteKit's other headers (2.61.1, adapter-node 5.5.4): `x-sveltekit-page` and `link` preloads (its own JS),
  no security headers. Assets: `immutable` for a year, precompressed gzip/Brotli.
- Known difference: SvelteKit's server render never marks the current category chip, because its relative link
  isn't equal to the path; hydration fixes it. Go marks it in the HTML, like the hydrated page.
- Routes: Go's `ServeMux` panics on overlapping patterns, so assets use `/assets/{file}` (Vite's output is flat)
  rather than `/assets/`, which would overlap `/{number}/{slug}`.
- Errors show the status and its standard text inside the layout, without the sidebar. Redirects have no body.
  An error after a handler started its response (usually a client that went away) is logged at debug level only.
- Every request is logged, assets and health checks included: kept on purpose.
  A trailing slash is a 308 to the path without it.
- Only pages with a sidebar load the archive and tag counts (`newSidebarPage`).
- JSON-LD is encoded with `json.Marshal`, which escapes `<`, `>`, and `&` (same JSON, safe inside `<script>`).

## Production data

- Backups: private repo `matteocontrini/notedimatteo-backups`, committed daily by a GitHub Action running
  `supabase db dump` (only the CLI: production is plain Postgres, not Supabase): `schema.sql`, `data.sql` (`COPY`), and an almost empty `roles.sql`.
- Import: create an empty database, then `psql "<url>" -v ON_ERROR_STOP=1 -f schema.sql -f data.sql`.
- The local clone `../notedimatteo-backups` stops at 2025-11-09. The 2026-11-20 backup (commit `73cc0e2`) is in
  the session scratchpad, under `prod-backup-73cc0e2`, which a reboot can clear: fetch it again if it's gone.
  The 2026-11-20 backup is imported locally as `notedimatteo_prod`; the original `notedimatteo` is unchanged.
- To use it, point `DATABASE_URL` in `.env` at `notedimatteo_prod`: `npm run dev` sources `.env`, which overrides
  a `DATABASE_URL` set on the command line.
- 2026-11-20: 670 posts (#1–#670), all published, no drafts; 99 tags; 1,095 post–tag links; all cached at
  renderer version 1. 189 posts use `[img]`, 43 contain backticks, 10 contain `<`, none have CRLF line endings,
  none contain `]]>`, and no two share a `publishedAt`.
- Images are served from `https://uploads.notedimatteo.it`.
- Production Postgres runs with `TimeZone = Etc/UTC`, so the Prisma 7 time zone bug doesn't affect the live site,
  and archive months were grouped in UTC (in Rome since 2026-11-23). Local comparisons force a UTC session to match.
- After a run against the production copy, restore its cache, which the server overwrites: extract the
  `post_html_cache` `COPY` block from `data.sql`, then `TRUNCATE post_html_cache` and load it in one transaction.
- `sessions` holds 11 production login tokens: treat the local copy like the backup repo.
