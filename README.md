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

After editing templates or SQL queries, regenerate the code:

```sh
go tool templ generate   # internal/views/*.templ
go tool sqlc generate    # internal/db/queries/*.sql
```

## Tests

```sh
npm test          # the editor's tests (Vitest)
npm run test:go   # the Go tests
```

The database tests run only when `TEST_DATABASE_URL` points at a database with the schema; they roll back what they write.

## Formatting and linting

Code formatting is handled by [Prettier](https://prettier.io/) and `gofmt`. To format the JavaScript, Svelte, and CSS files:

```sh
npm run format
```

To check formatting and run [ESLint](https://eslint.org/) together:

```sh
npm run lint
```

TypeScript, accessibility, and Svelte compiler errors are caught by [`svelte-check`](https://www.npmjs.com/package/svelte-check):

```sh
npm run check
```

Editor configuration is committed for both VS Code (`.vscode/`) and WebStorm (`.idea/`), so Prettier and ESLint run without manual setup. Both need a custom file pattern because the IDE defaults omit `.svelte`.
