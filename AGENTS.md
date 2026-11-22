`docs/ARCHITECTURE.md` describes how the site works.

Verifying changes:

- Run `npm run format` to format your changes before finishing.
- Run `npm run check` for formatting, ESLint, svelte-check, and `go vet`.
- Run `npm test` for the editor, Go, and browser tests. The Go and browser tests need Docker.

Both scripts first regenerate the templ and sqlc code and build the assets.
