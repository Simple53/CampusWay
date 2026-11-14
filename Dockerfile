# get modules, if they don't change the cache can be used for faster builds
FROM golang:1.25 AS base
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ARG COMMIT_FLAG

WORKDIR /src
COPY go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# build the application
FROM base AS build
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o /app/campusway -trimpath -ldflags "-s -w -buildid= ${COMMIT_FLAG}" .

# Import the binary from build stage
FROM gcr.io/distroless/static AS prd
WORKDIR /home/nonroot
COPY --from=build /app/campusway /home/nonroot
ENTRYPOINT ["/home/nonroot/campusway", "-config", "/home/nonroot/config.toml"]
