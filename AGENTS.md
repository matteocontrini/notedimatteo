Verifying changes:

- Run `npm run format` to format your changes with Prettier before finishing.
- Run `npm run lint` to check formatting and ESLint together.
- Run `npm run check` to catch TypeScript, a11y, and Svelte compiler errors.
- Run `gofmt -l cmd internal web` and `go tool templ fmt internal/views` for Go and templ formatting.
- After editing a `.templ` file, run `go tool templ generate`. After editing `internal/db/queries/*.sql`, run
  `go tool sqlc generate`. Generated files are commited.
- Run `npm test` for the editor's tests and `npm run test:go` for the Go tests.
