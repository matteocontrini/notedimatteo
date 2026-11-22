# Redesign plan

Merge the blog (matteosonoio.it, Hugo) into this site with a new design, and keep every revision of a post.
It builds on the Go server: `go-migration-plan.md` (done 2026-11-21) and `ARCHITECTURE.md`.

Working notes and open questions: `redesign-notes.md`.

## Phases

Each phase reaches production on its own. Phases 2–4 can happen in any order.

1. **Revisions:** keep every body revision of a post.
2. **New design:** the notes site in the new design, changing only templates and CSS.
3. **Articles:** import the blog, bilingual pages, domains and redirects.
4. **New editor:** CodeMirror 6 with live preview.

Design reference: the [design canvas](https://claude.ai/artifact/DeA9bBLoajjHfTZfgk1uU2),
boards E2/E4 (home), D2 (notes), D3 (article), D4 (article archive), F2 (projects),
and G1–G3 (editor).

## Decisions

| Area  | Choice                                                                                                    |
|-------|-----------------------------------------------------------------------------------------------------------|
| URLs  | Still built only in `internal/urls`; the merged site's URL scheme is decided in phase 3                    |
| CSS   | From phase 2: only design tokens, no arbitrary values, and a Go test that checks component classes        |
| Dates | English month names in phase 3, next to the Italian ones                                                  |

## Phase 1: Revisions

- [ ] Add a revisions table keyed by post and `bodyRevision`
- [ ] Save a revision in the save transaction whenever the body changes
- [ ] List revisions on the edit page and restore one

## Phase 2: New design

- [ ] Reset the Tailwind theme to the design tokens: colors, fonts, and spacing
- [ ] Self-host Public Sans and Source Serif 4
- [ ] Add a Go test that fails on arbitrary values and on template classes missing from the built CSS
- [ ] Move repeated patterns into template partials or `@utility` rules
- [ ] Document the tokens and partials in `AGENTS.md`
- [ ] Redesign the layout, home, notes, archive, search, and about pages; the home has no articles until phase 3

## Phase 3: Articles

- [ ] Decide the merged site's URL scheme and domains
- [ ] Reorganize `internal/server` before adding article pages: group the page handlers, by feature or in packages
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

## Phase 4: New editor

- [ ] Replace the textarea with CodeMirror 6 live preview, still storing plain Markdown
- [ ] Add the quick composer on the notes page (G1)
- [ ] Add the full editor with the details sidebar (G2)
- [ ] Add the drafts page (G3)
- [ ] Add the `/` menu, moving blocks, and image paste and drag
- [ ] Fetch page titles for pasted links, refusing private network addresses
- [ ] Add note links (`#670`) and footnotes to the renderer
