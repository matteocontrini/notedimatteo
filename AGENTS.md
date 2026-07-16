- Use Svelte 5 syntax.
- Do not commit unless explicitly asked.
- If the user makes changes to the code, do not revert them to your previous generation.

The code must pass formatting, linting, and Svelte checks before committing:

- Run `npm run format` to format your changes with Prettier before finishing.
- Run `npm run lint` to check formatting and ESLint together.
- Run `npm run check` to catch TypeScript, a11y, and Svelte compiler errors.
