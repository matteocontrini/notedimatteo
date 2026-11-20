FROM node:24-trixie-slim AS assets
WORKDIR /app
COPY package.json package-lock.json .npmrc ./
# The prepare script is for the SvelteKit app and isn't needed for this build
RUN npm ci --ignore-scripts
COPY vite.go.config.ts ./
COPY src/app.css src/app.css
# Tailwind scans the templ components for class names
COPY internal/views internal/views
COPY web web
RUN npm run go:assets

FROM golang:1.26-trixie AS build
RUN apt-get update && apt-get install -y --no-install-recommends libvips-dev pkg-config
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
COPY web web
COPY --from=assets /app/web/dist web/dist
RUN go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

FROM debian:trixie-slim AS runtime
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates libvips42t64 \
	&& rm -rf /var/lib/apt/lists/*

# Proves that govips works against the runtime libvips: docker build --target vipscheck .
FROM build AS vipscheck-build
RUN go build -tags vipscheck -o /out/vipscheck ./cmd/vipscheck

FROM runtime AS vipscheck
COPY --from=vipscheck-build /out/vipscheck /usr/local/bin/vipscheck
RUN vipscheck

FROM runtime AS server
COPY --from=build /out/server /usr/local/bin/server
USER nobody
ENTRYPOINT ["server"]
