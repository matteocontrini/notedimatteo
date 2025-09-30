# Note di Matteo

This repository contains the source code for the website [notedimatteo.it](https://notedimatteo.it).

The project is built with SvelteKit (full stack with SSR) and uses a PostgreSQL database.

> [!CAUTION]
> This project is largely vibe-coded, read at your own risk and don't judge :)

## Configuration

Refer to `.env.example` for the required environment variables. Create a `.env` file in the root directory and set the variables accordingly.

## Development

Install dependencies with `npm install` and start a development server:

```sh
npm run dev

# or start the server and open the app in a new browser tab
npm run dev -- --open
```

## Production build

To build the production version:

```sh
npm run build
```

Then run:

```sh
npm run migrate
npm run start
```
