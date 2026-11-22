# Redesign plan

Merge the blog (matteosonoio.it, Hugo) into this site with a new design, and keep every revision of a post.
It builds on the Go server: `go-migration-plan.md` (done 2026-11-21) and `ARCHITECTURE.md`.

Working notes and open questions: `redesign-notes.md`.

## Phases

Each phase reaches production on its own. Phase 4 comes before phase 5; the others can happen in any order.

1. **CodeMirror:** CodeMirror 6 with live preview replaces the body textarea on today's edit page.
2. **Revisions:** keep every body revision of a post.
3. **New design:** the notes site in the new design, changing only templates and CSS.
4. **Image proxy:** imgproxy generates image sizes and formats on request, and the Go server drops libvips.
5. **Articles:** import the blog, bilingual pages, domains and redirects.
6. **New editor:** the article editor, the quick composer, and new editing features.

Design reference: the [design canvas](https://claude.ai/artifact/DeA9bBLoajjHfTZfgk1uU2),
boards E2/E4 (home), D2 (notes), D3 (article), D4 (article archive), F2 (projects),
and G1–G3 (editor).

## Decisions

| Area  | Choice                                                                                                    |
|-------|-----------------------------------------------------------------------------------------------------------|
| URLs  | Still built only in `internal/urls`; the merged site's URL scheme is decided in phase 5                    |
| CSS   | From phase 3: only design tokens, no arbitrary values, and a Go test that checks component classes        |
| Dates | English month names in phase 5, next to the Italian ones                                                  |

## Phase 1: CodeMirror

Only the body field changes; the rest of the edit page stays as it is.

- [x] Replace the body textarea with CodeMirror 6, still storing plain Markdown; keep focus on load, the height
  following the content, line wrapping, spellcheck, and the undo history across ⌘P
- [x] Turn the Markdown helpers into CodeMirror commands for today's shortcuts: ⌘B, ⌘I, ⌘⇧' (quote), ⌘K (link);
  ⌘P, ⌘S, and ⌘⇧S stay on the window
- [x] Upload pasted images and the upload button's file at the cursor, through a placeholder the upload replaces
- [x] Style the Markdown: emphasis, links, headings, quotes, code; markers stay visible but dimmed
- [x] Show uploaded images under their `[img]` line, at their size
- [x] Theme it with the site's font and colors, light and dark
- [x] Update the browser tests, which type into `textarea[name=body]`
- [x] Store bodies with `\n` line breaks, and convert the ones saved with `\r\n` since cutover (`00003`)
- [ ] Check it by hand on Chrome and Firefox for Android

## Phase 2: Revisions

- [ ] Add a revisions table keyed by post and `bodyRevision`
- [ ] Save a revision in the save transaction whenever the body changes
- [ ] List revisions on the edit page and restore one

## Phase 3: New design

- [ ] Reset the Tailwind theme to the design tokens: colors, fonts, and spacing
- [ ] Self-host Public Sans and Source Serif 4
- [ ] Add a Go test that fails on arbitrary values and on template classes missing from the built CSS
- [ ] Move repeated patterns into template partials or `utility` rules
- [ ] Document the tokens and partials in `AGENTS.md`
- [ ] Redesign the layout, home, notes, archive, search, and about pages; the home has no articles until phase 5

## Phase 4: Image proxy

- [ ] Choose the CDN in front of imgproxy (Railway's edge or Bunny), and how pages pick formats: in the URL
  with `<picture>`, or by `Accept`
- [ ] Deploy imgproxy as a second Railway service that reads the originals from R2, with signed URLs
- [ ] Compare imgproxy's JPEGs with today's mozjpeg files
- [ ] Store uploads as originals only; read their width and height in Go, with the EXIF orientation
- [ ] Render `[img]` with signed imgproxy URLs and `srcset`, and bump `markdown.Version`
- [ ] Remove libvips from the Go server: govips, cgo, sharp's npm packages, `.libvips`, and the `libvips-test` stage

## Phase 5: Articles

- [ ] Decide the merged site's URL scheme and domains
- [ ] Reorganize `internal/server` before adding article pages: group the page handlers, by feature or in packages
- [ ] Add article fields: type, language, summary, last-updated date, table of contents and comments flags
- [ ] Import the Hugo posts: front matter, `fig`/`relref`/`center` shortcodes, raw HTML (`<sup>`, `<video>`)
- [ ] Upload the blog's images and videos to S3; imgproxy serves their sizes and WebP versions
- [ ] Enable raw HTML, Chroma syntax highlighting, footnotes, and tables for articles
- [ ] Add article, article archive, and projects pages
- [ ] Add Italian/English interface strings, English dates, and `hreflang` links; notes stay Italian-only
- [ ] Add related notes on articles, by tag
- [ ] Add giscus comments, keeping article paths so the `pathname` mapping still finds threads
- [ ] Add feeds for notes, articles, and everything
- [ ] Redirect old blog URLs, Hugo feeds, and `blog.contrini.it`
- [ ] Retire Hugo and Netlify

## Phase 6: New editor

- [ ] Add the quick composer on the notes page (G1)
- [ ] Add the full editor for articles, with the details sidebar (G2)
- [ ] Add the drafts page (G3)
- [ ] Add the `/` menu, moving blocks, and image drag
- [ ] Fetch page titles for pasted links, refusing private network addresses
- [ ] Add note links (`#670`) and footnotes to the renderer
