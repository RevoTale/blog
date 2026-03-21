FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build hashed + minified static assets during image build.
RUN GOCACHE=/tmp/go-cache GOMODCACHE=/go/pkg/mod \
    go run ./framework/cmd/staticassetsgen \
      -source internal/web/static \
      -out internal/web/static-build \
      -manifest internal/web/static-build/manifest.json \
      -url-prefix /.revotale/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags='-s -w' -o /out/blog .

FROM gcr.io/distroless/static-debian12 AS runtime

WORKDIR /app

COPY --from=builder /out/blog /app/blog
COPY --from=builder /src/internal/web/static-build /app/internal/web/static-build
COPY --from=builder /src/internal/web/public /app/internal/web/public

ENV BLOG_LISTEN_ADDR=:8080
ENV BLOG_STATIC_BUILD_DIR=/app/internal/web/static-build
ENV BLOG_STATIC_MANIFEST_PATH=/app/internal/web/static-build/manifest.json
ENV BLOG_PUBLIC_DIR=/app/internal/web/public

EXPOSE 8080

ENTRYPOINT ["/app/blog"]
