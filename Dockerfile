FROM golang:1.27.0-alpine3.23 AS builder

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

FROM gcr.io/distroless/static-debian13:nonroot AS runtime

WORKDIR /app

COPY --from=builder /out/blog /app/blog
COPY --from=builder /src/web/assets-build /app/web/assets-build
COPY --from=builder /src/web/public /app/web/public

ENV BLOG_LISTEN_ADDR=:8080

EXPOSE 8080

ENTRYPOINT ["/app/blog"]
