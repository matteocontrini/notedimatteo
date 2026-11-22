# Note di Matteo

This repository contains the source code for the website [notedimatteo.it](https://notedimatteo.it).

## Project architecture

Following the migration from SvelteKit, this project is now a Go application. A Go server renders the pages with [templ](https://templ.guide), reading and writing posts from a PostgreSQL instance. 

The post editor is a Svelte 5 app (no SvelteKit) that the HTML template mounts on the edit page. CSS and scripts are bundled by Vite and embedded in the Go binary.

Architecture details are described in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

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

The database schema is managed with [goose](https://github.com/pressly/goose) migrations in `internal/db/migrations`. Migrations are applied by the Go server at startup.

Editor configuration is committed for both VS Code (`.vscode/`) and JetBrains IDEs (`.idea/`), so Prettier and ESLint should work without manual setup.

## Scripts

```sh
npm run generate   # generates the templ and sqlc code, and builds the assets
npm run format     # formats with Prettier, gofmt, and templ fmt
npm run check      # checks formatting, runs ESLint, svelte-check, and go vet
npm test           # runs the editor's tests (Vitest), the Go tests, and the browser tests (Playwright)
```

Additional facts:
- The scripts `dev`, `check`, and `test` also run `generate` first.
- Image processing is handled by libvips, installed with npm. When you run `npm install`, the file `.libvips/vips.pc` is written and it points the Go build at the libvips in `node_modules`. Go commands that compile the image processing thefore need `PKG_CONFIG_LIBDIR="$PWD/.libvips"` (already included in the scripts).
- Docker is needed to run the tests with Testcontainers.
