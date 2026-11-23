FROM node:26-trixie-slim AS assets
WORKDIR /app
COPY package.json package-lock.json .npmrc ./
# The postinstall script runs below, once its file is copied; no other package
# needs its install scripts for this build
RUN npm ci --ignore-scripts
# npm installs sharp's prebuilt libvips for this architecture; the script
# writes the pkg-config file that points the Go build at it
COPY scripts/libvips.mjs scripts/
RUN node scripts/libvips.mjs
COPY vite.config.ts ./
# Tailwind scans the templ components for class names
COPY internal/views internal/views
COPY web web
RUN npx vite build

FROM golang:1.27-trixie AS build
RUN apt-get update && apt-get install -y --no-install-recommends pkg-config
WORKDIR /app
# Same paths as in the assets stage, which the pkg-config file refers to
COPY --from=assets /app/node_modules/@img node_modules/@img
COPY --from=assets /app/.libvips .libvips
ENV PKG_CONFIG_LIBDIR=/app/.libvips
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
COPY web web
COPY --from=assets /app/web/dist web/dist
RUN go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

FROM debian:trixie-slim AS runtime
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*
# sharp's libvips is a single library with its dependencies built in
COPY --from=assets /app/node_modules/@img/sharp-libvips-linux-*/lib/libvips-cpp.so.* /usr/local/lib/
RUN ldconfig
# Limit glibc to two memory pools, otherwise it creates one per thread and
# memory grows with each upload
ENV MALLOC_ARENA_MAX=2

# Runs the image processing tests in the runtime image: docker build --target libvips-test .
FROM build AS libvips-test-build
RUN go test -c -o /out/libvips.test ./internal/images/libvips

FROM runtime AS libvips-test
COPY --from=libvips-test-build /out/libvips.test /usr/local/bin/libvips.test
RUN libvips.test

FROM runtime AS server
COPY --from=build /out/server /usr/local/bin/server

# Litestream wraps the Go app and backs the SQLite database (see litestream.yml)
COPY --from=litestream/litestream:0.5.18 /usr/local/bin/litestream /usr/local/bin/litestream
COPY litestream.yml /etc/litestream.yml

USER nobody
ENTRYPOINT ["litestream", "replicate"]
