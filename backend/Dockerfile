# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS base
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum* ./
COPY *.go schema.sql ./
# go.sum is generated here so the first build needs no local Go toolchain.
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy

# `docker build --target test ./backend` runs vet + unit tests.
FROM base AS test
RUN --mount=type=cache,target=/go/pkg/mod go vet ./... && go test -count=1 ./...

FROM base AS build
RUN --mount=type=cache,target=/go/pkg/mod go build -trimpath -ldflags="-s -w" -o /out/cypherdoc .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/cypherdoc /app/cypherdoc
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/cypherdoc"]
