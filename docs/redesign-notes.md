# Redesign notes

Working notes for `redesign-plan.md`: facts found while working, gotchas, and open questions. Read both
before resuming. Record what the code and git history don't show; remove notes that stop being true.
`go-migration-notes.md` has the production data copy, backups, and other facts from the migration.

## Working agreements

- The user handles git: never commit, push, or stage.
- Plan terminology: phases are the broad units, steps are the smaller ones inside a phase.
- Keep plan and notes succinct and schematic.

## Open questions

- Merged site's URL scheme and domains (phase 5). Affects routes, redirects, and giscus threads.
- English home: a variant of board E3, not designed yet. Italian home: E2 or E4.

## Image proxy (phase 4)

- Uploads store only the original; imgproxy (libvips-based, its own Railway service) resizes and encodes on
  request, and a CDN caches each version. imgproxy has no cache of its own.
- URLs carry an HMAC signature with a key that only Go and imgproxy share, so nobody else can request
  arbitrary sizes.
- Upload still needs width and height for `[img]`: `image.DecodeConfig` reads them without decoding pixels;
  phone photos also need the EXIF orientation, which can swap them. imgproxy's info endpoint is probably paid.
- Existing posts (about 190 use `[img]`): their `.jpg` files are already re-encoded originals and can be
  imgproxy's sources; keep the `_preview.jpg` files for feed readers that cached them.
- Formats by `Accept` need the CDN to cache one copy per format; naming the format in the URL with `<picture>`
  avoids that.
- To check: imgproxy's official image is probably built with mozjpeg; compare its output with today's files.
- Libvips memory spikes move from the Go server to imgproxy: give that service a memory budget.
- Alternatives without a service to run: Bunny Optimizer, Cloudflare Images (monthly fee).

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
