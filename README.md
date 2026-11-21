# Note di Matteo

This repository contains the source code for the website [notedimatteo.it](https://notedimatteo.it).

## Project architecture

- A Go server renders the pages with [templ](https://templ.guide), with posts stored in PostgreSQL.
- The post editor is a Svelte 5 app that Go mounts on the edit page.
- Vite builds the CSS (Tailwind) and the scripts, and the Go binary embeds them.

## Configuration

Refer to `.env.example` for the environment variables. Create a `.env` file in the root directory and set the variables accordingly.

The `STORAGE_S3_*` variables are optional: without them, image uploads are disabled.

## Development

You need Go and Node.js.

Install dependencies with `npm install` and start the server on port 6200:

```sh
npm run dev
```

There's no hot reload: restart the server after changing templates, CSS, or scripts.

The database schema lives in [goose](https://github.com/pressly/goose) migrations in `internal/db/migrations`. The server applies new migrations when it starts.

## Scripts

```sh
npm run generate   # generates the templ and sqlc code, and builds the assets
npm run format     # formats with Prettier, gofmt, and templ fmt
npm run check      # checks formatting, runs ESLint, svelte-check, and go vet
npm test           # runs the editor's tests (Vitest), the Go tests, and the browser tests (Playwright)
```

`dev`, `check`, and the Go and browser tests run `generate` first. Generated files are committed. `npm test` runs `test:js`, `test:go`, and `test:e2e`, which also run on their own.

`npm install` writes `.libvips/vips.pc`, which points the Go build at the libvips in `node_modules`. Go commands that compile the image processing need `PKG_CONFIG_LIBDIR="$PWD/.libvips"`: the scripts set it, and the editor's Go settings need it too.

The Go and browser tests start a throwaway PostgreSQL (and an S3 server for the browser tests) with Docker, through [Testcontainers](https://testcontainers.com), so Docker must be running; `go test -short ./...` skips the database tests. To run them against another database as it is, such as a copy of production data, set `TEST_DATABASE_URL`: the tests that write roll back their changes.

Editor configuration is committed for both VS Code (`.vscode/`) and WebStorm (`.idea/`), so Prettier and ESLint run without manual setup.
