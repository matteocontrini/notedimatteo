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

## CodeMirror (phase 1)

- Decided 2026-11-22: markers (`**`, `#`, `>`, link brackets) stay visible but dimmed, not hidden away from the
  cursor; the upload button inserts at the cursor like paste; `[img]` lines show their image below; Android
  (Chrome, Firefox) is used rarely but should work.
- `@codemirror/lang-markdown` wraps the `@lezer/markdown` parser as a CodeMirror language. Don't use its
  `markdown()`: it pulls in `lang-html`, which brings the HTML, CSS, and JavaScript parsers even with
  `completeHTMLTags: false` (measured with esbuild, minified: core 273 KB / 89 KB gzipped; plus `markdown()`
  504 / 174; plus `commonmarkLanguage` only 308 / 100). `commonmarkLanguage` is a plain `Language`, with no
  `configure`: `body/language.ts` builds `new Language(commonmarkLanguage.data, parser.configure([Strikethrough]))`,
  as `markdown()` does inside.
- The editor bundle went from 84 KB (31 KB gzipped) to 401 KB (135 KB gzipped); only the edit page loads it.
- `@lezer/markdown` tags every marker (`HeaderMark QuoteMark ListMark LinkMark EmphasisMark CodeMark`) as
  `tags.processingInstruction`: dimming them is one rule in a `HighlightStyle`.
- Its `Strikethrough` needs `~~`; goldmark also accepts `~`, which the editor won't style.
- `defaultKeymap` binds Mod-i to `selectParentSyntax`: our keymap needs a higher precedence.
- Block widgets (the images under `[img]`) must come from a `StateField`: a `ViewPlugin` that provides them
  throws "Block decorations may not be specified via plugins".
- The parser reads `[img …]` and `[/img]` as link brackets; a mark decoration dims them, and its CSS also covers
  the highlighter's spans inside it.
- In a `HighlightStyle`, the later rule wins when two tags apply: the marker rule is last, so markers inside links
  and quotes are dimmed too.
- CodeMirror injects its styles outside CSS cascade layers, so they beat Tailwind's layered rules: the editor's
  theme is an `EditorView.theme` that reads `--editor-*` variables, which `main.css` sets for light and dark.
  Tailwind emits a `--color-*` variable only when something uses it; those variables do.
- The shortcuts compare `event.key` like the textarea's handler did (so every keyboard layout keeps the same
  keys), in a `keydown` handler at `Prec.highest`, ahead of the default keymap.
- The editor should look like the published page (decided 2026-11-22). Matched: images as wide as the text,
  quotes (border; citations show the quotation marks from `--quote-open-mark` and `--quote-close-mark`, which
  `main.css` shares with the site), underlined link text, bold at 700, inline code at full size, code blocks on a
  grey background. Left on purpose: headings (posts show them as plain text: Tailwind's reset removes their
  styles, and the site never adds them back), paragraph spacing, list bullets, and single line breaks, which are
  what plain-text editing looks like. Phase 3 changes the published page: restyle the editor with it.
- Paragraphs (2026-11-22, a first step towards G2, where blocks are separated by space and no blank lines show):
  - Which blank lines hide depends only on the text (decided 2026-11-22, after a cursor-based rule made empty
    paragraphs vanish when the cursor left them): in a run of blank lines, the lines next to a block are the space
    between blocks, and the lines in between alternate between an empty paragraph (shown as an empty line) and the
    space after it (`hiddenLinesOfRun`). One or two blank lines between blocks are a paragraph break; three are an
    empty paragraph. Blank lines in code blocks always show; a blank line with a cursor shows only as a fallback
    for odd texts like a body ending in `A\n`.
  - A state field replaces each hidden run with a block widget (10 px between blocks, 0 at the start and end) and
    makes the run plus the line breaks around it atomic: the arrows cross a paragraph break in one step, and
    Backspace and Delete there join the paragraphs (word-processor style: "AB") or remove an empty paragraph.
  - Enter: lists and quotes go through lang-markdown's `insertNewlineContinueMarkupCommand({nonTightLists: false})`
    (its default turns a two-item list loose instead of leaving it); leaving a list adds the blank line, or the
    next text would join the last item. Elsewhere Enter inserts `\n\n`; on a blank line it only adds a missing
    separation; in code blocks it falls through to the plain line break. Shift+Enter inserts one line break.
  - Enter always inserts `\n\n` and moves the cursor after it, as in a word processor: at the start of a paragraph
    the paragraph moves down and an empty one appears above; on an empty paragraph it adds another. On a blank line
    right below a block it only adds the missing blank line. In code blocks Enter keeps the indentation of
    the line above (`insertNewlineAndIndent`); a closing fence typed after an indented line needs the indentation
    removed first, since a fence indented by 4+ spaces doesn't close the block.
  - Code block lines use `text-sm` like posts: 0.875rem with a 1.25rem line height.
  - In code blocks, Tab inserts 4 spaces at the cursor or indents the selected lines, and Shift+Tab removes a step
    (`indentCode`, `dedentCode` in `blocks.ts`). Elsewhere Tab keeps moving the focus. Posts' code blocks mix tabs, 2
    spaces, and 4 spaces (22 blocks in production, 2026-11-22); tabs would show 8 columns wide on the site.
  - `testing.ts` marks cursors with `|` and selections with `«` and `»`.
  - A fenced block without its closing fence runs to the end of the text, so "inside a code block" can't be
    `pos < node.to`: `insideCodeBlock` checks the closing `CodeMark`.
  - An image inserted at the end of the text gets a paragraph after it, where the cursor goes.
  - Block replace decorations default to inclusive, which makes them valid on an empty line.
  - Importing `insertNewlineContinueMarkupCommand` brings lang-markdown's GFM parser setup: +13 KB (4.5 KB gzipped).
  - Single line breaks still show as new lines; the site joins them into one paragraph. Lines in a quote continue
    with `> `, which the site also joins.
- The editor lives in `web/editor/body`, one module per part with its tests; `testing.ts` writes cursors as `|`
  in the text, so command tests read as before and after. `tests/e2e/editing.spec.ts` checks paragraphs in the
  browser against the server's preview.
- Quotes and code blocks get line classes from the parse tree (`blockDecorations`, a `ViewPlugin`). The category
  reaches the editor as `data-category` on its container, as on the site's `<article>`. Lines take horizontal
  margins only: CodeMirror measures line heights, which vertical margins would break.
- Uploads show an "Uploading image…" widget at the cursor; a `StateField` maps its position through edits, and the
  markup replaces it when the upload ends. The cursor starts at the end of the body.
- Bug found 2026-11-22: browsers encode `FormData` text with `\r\n` line breaks, and the Go save endpoint stored
  them (SvelteKit stored `\n`: no production post had `\r` before cutover). The textarea hid it; CodeMirror turns
  `\r\n` into `\n`, so a cursor at the string's length fell outside the document. Fixed in `parsePostForm` and
  in the editor; `00003_body_line_breaks.sql` converts the bodies saved since cutover.
- Browser tests find the editor as `getByRole('textbox', {name: 'Body'})` and read it with `toHaveText`, which
  works because the markers stay visible.
- rclone's `serve http` caches directory listings for 5 minutes, so a second upload into the same month's folder
  answered 404 to the browser; `--dir-cache-time 0s` turns the cache off in the browser tests.
- The project needs Node 26 (`.nvmrc`); this machine's default is 24, so prefix commands with nvm's 26 on `PATH`.
- `markdown()`'s `markdownKeymap` (lists continue on Enter) and `pasteURLAsLink` come only with `markdown()`;
  the textarea has neither, so the editor starts without them.
- Rejected: OverType (a real textarea under a styled overlay, best on mobile, but its README rules out images and
  different font sizes), ink-mde (last release September 2024; pulls KaTeX, Vim mode, SolidJS, every code
  language), ProseMirror editors like Milkdown and TipTap (they store a document model, not Markdown).
- `@codemirror/view` has specific handling for Android and Gboard input, with many fixes in its changelog.

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
