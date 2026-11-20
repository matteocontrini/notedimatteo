# Go migration notes

Working notes for `GO-MIGRATION-PLAN.md`: facts found while working, gotchas, and
open questions. Read both before resuming. Record what the code and git history
don't show; remove notes that stop being true.

## Working agreements

- The user handles git: never commit, push, or stage.
- Plan terminology: phases are the broad units, steps are the smaller ones inside a phase.
- Keep plan and notes succinct and schematic.

## Open questions

- Merged site's URL scheme and domains (phase 4). Affects routes, redirects, and giscus threads.
- JPEG encoding: accept Debian's libvips without mozjpeg, or build libvips with mozjpeg (phase 1, step 4).
- English home: a variant of board E3, not designed yet. Italian home: E2 or E4.
- Production `BODY_SIZE_LIMIT`: `.env.example` has `1M`, local `.env` has `20M`. Match it in Go's upload limit.
- Production database `TimeZone` (`SHOW timezone`): if it isn't UTC, the live SvelteKit app has shown wrong times
  since the Prisma 7 upgrade (see below), and the archive's month grouping depends on it too.

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
- `web/dist` is gitignored and embedded: run `npm run go:assets` before `go build`, `go test`, or `go vet`.
- `go.mod` has `ignore ./node_modules`: the `flatted` npm package ships a Go package that `./...` would otherwise pick up.
- Check gofmt with `gofmt -l cmd internal web`, not `.` (avoids `node_modules`).
- libvips isn't installed on the Mac; `govips` is only compiled in Docker until phase 1 step 4 (then `brew install vips`).
- Local database: 31 posts, 18 published, all published ones cached at renderer version 1.
  Posts 1–5 share one `publishedAt` (seed data); queries order by `publishedAt DESC, number DESC`.
- Local Postgres runs with `TimeZone = Europe/Rome`.
- Comparing with SvelteKit:
  - The user may have `go:dev` running on 6200: run test servers on 6201, built to the scratchpad and started
    directly, and stop them only by their own PIDs. Never `pkill -f` patterns that match the user's servers.
  - Start SvelteKit with `node node_modules/vite/bin/vite.js dev` so its PID is the server's.
  - Add `options=-c%20TimeZone%3DUTC` to both apps' `DATABASE_URL`, or SvelteKit shows wrong times locally.
  - Drive headless Chrome over the DevTools protocol (Node's built-in `WebSocket`, `--remote-debugging-port`):
    compare `document.body.innerText`, emulate `prefers-color-scheme`, click controls, take screenshots.
- Expected differences: ties are ordered deterministically in Go (posts 1–5 by number, sidebar tags with equal
  counts by name).

## Docker and deployment

- Stages: `assets` (Node) → `build` (Go + libvips-dev) → `runtime` (Debian trixie + libvips) → `server` (last, the default).
  The `assets` stage copies `internal/views` so Tailwind can scan the components; `build` uses the committed `_templ.go` files.
- `docker build --target vipscheck .` builds and runs `cmd/vipscheck` against the runtime libvips.
- Image is about 242 MB, runs as `nobody`; `.dockerignore` keeps `.env` and `node_modules` out.
- Railway probably builds a root `Dockerfile` automatically (unverified): don't merge `go-migration` into `main` before cutover.
- Production currently starts through `npm start`, whose launcher process used 55.5 MiB.

## govips and libvips

- Install with `go get github.com/davidbyttow/govips/v2/vips@v2.20.0`. Asking for `.../govips/v2` resolves to a stale 2020 root-module version.
- `vips.Startup(&vips.Config{})` turns caching off: values ≥ 0 are applied. `Startup(nil)` uses govips defaults, with caching.
- Debian trixie ships libvips 8.16.1, which ignores `trellis_quant`, `overshoot_deringing`, `optimize_scans`, and `quant_table`: no mozjpeg.

## Current SvelteKit behavior to preserve

- Markdown (unified/remark): raw HTML is dropped (inline tags removed, text kept); `&` becomes `&#x26;`; `<br>`;
  single and double `~` strike through; no tables or autolinks. `[img]` is handled on text nodes after parsing.
- `[img width=W height=H link=URL alt=…]path[/img]`: `src` is the `_preview` variant, wrapped in a link to the
  original (or `link`), `loading="lazy"`.
- Uploads: keys `YYYY/MM/<12 hex>.jpg` and `…_preview.jpg` (server-local month); main JPEG q85, preview 1200 px
  wide q80, both mozjpeg with auto-orientation; the response carries the preview's dimensions; S3 metadata `original`.
- Sessions: cookie `notedimatteo_session`, 60 days, `httpOnly`, `sameSite=lax`, `secure` outside dev;
  `lastActivityAt` updated at most once a minute and has no database default (Prisma `@updatedAt`), so inserts set it.
- Search: Prisma `contains` with `mode: insensitive` doesn't escape `%` or `_`.
- Prisma 7's `@prisma/adapter-pg` replaces the offset of every `timestamptz` it reads with `+00:00`, so times are
  only right when the session time zone is UTC. Go reads them correctly.
- Archive months use `EXTRACT` in the database session's time zone, not Rome's; Go keeps the same SQL.
- Tags inside a post have no order column and no `ORDER BY`, in both apps: whatever order Postgres returns,
  in practice the order saved from the editor. Kept as is on purpose; not a bug to fix during the migration.
- `?page=` that isn't a positive integer: SvelteKit passes it to Prisma (likely an error); Go shows page 1.
- Theme: mode-watcher stores the preference in localStorage `mode-watcher-mode` (`system`/`light`/`dark`); dark mode is a `.dark` class.
  Go applies it with an inline head script (`theme-init.gohtml`) and handles the select in `web/assets/site.ts`.
- `https://notedimatteo.it` is hard-coded in `MetaTags.svelte` (canonical) and the RSS feed.
- GoatCounter is disabled when the host isn't `notedimatteo.it`.

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
