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

## Local development

- SvelteKit: `npm run dev`, port 6100. Go: `npm run go:dev`, port 6200 (builds assets, sources `.env`, `go run`).
- No hot reload: restart `go:dev` after template or CSS changes.
- `web/dist` is gitignored and embedded: run `npm run go:assets` before `go build`, `go test`, or `go vet`.
- `go.mod` has `ignore ./node_modules`: the `flatted` npm package ships a Go package that `./...` would otherwise pick up.
- Check gofmt with `gofmt -l cmd internal web`, not `.` (avoids `node_modules`).
- libvips isn't installed on the Mac; `govips` is only compiled in Docker until phase 1 step 4 (then `brew install vips`).
- Local database: 31 posts, 18 published, all published ones cached at renderer version 1.
  Posts 1–5 share one `publishedAt` (seed data); queries order by `publishedAt DESC, number DESC`.
- Comparing with SvelteKit: run both, diff `.post-content` blocks, and take headless Chrome screenshots
  (`"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --screenshot`).
  Headless Chrome renders SvelteKit in dark mode; the Go skeleton has no theme script yet.

## Docker and deployment

- Stages: `assets` (Node) → `build` (Go + libvips-dev) → `runtime` (Debian trixie + libvips) → `server` (last, the default).
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
- Theme: mode-watcher stores the preference in localStorage `mode-watcher-mode` (`system`/`light`/`dark`); dark mode is a `.dark` class.
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
