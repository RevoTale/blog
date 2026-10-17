FROM golang:1.27.2-alpine3.23@sha256:9e45f0eb4a63ed37ad6e604950407558b7c738e34e015ae77a5d4ad369cde079 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build hashed + minified static assets during image build.
RUN GOCACHE=/tmp/go-cache GOMODCACHE=/go/pkg/mod \
    go tool no-js gen assets -root .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags='-s -w' -o /out/blog ./cmd/server

FROM gcr.io/distroless/static-debian13:nonroot@sha256:1c2c046bc09ed40fad370b599a0b1ae7987f55b01e247cf27a7c27cd97e5bbc7 AS runtime

WORKDIR /app

COPY --from=builder /out/blog /app/blog
COPY --from=builder /src/web/assets-build /app/web/assets-build
COPY --from=builder /src/web/public /app/web/public

ENV BLOG_LISTEN_ADDR=:8080

EXPOSE 8080

ENTRYPOINT ["/app/blog"]
