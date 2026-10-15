FROM golang:1.27.2-alpine AS builder

ENV GOOS=linux
ENV CGO_CFLAGS_ALLOW="-Xpreprocessor"

RUN apk upgrade --no-cache && apk add --no-cache gcc g++ vips-dev vips-heif libheif-aom libheif-dav1d libheif-libde265 libheif-x265 git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -o /build/app -ldflags="-s -w" .

FROM builder AS test
RUN apk add --no-cache ffmpeg
RUN go test -race ./... && go vet ./...

FROM alpine:latest AS runtime

RUN apk upgrade --no-cache && apk --no-cache add ca-certificates ffmpeg vips vips-heif libheif-aom libheif-dav1d libheif-libde265 libheif-x265 && addgroup -g 1000 pepic && adduser -D -u 1000 -G pepic pepic
COPY --from=builder /build/app /app/pepic
COPY html /app/html
COPY static /app/static
COPY etc/pepic /etc/pepic
RUN mkdir -p /app/uploads && chown pepic:pepic /app/uploads
WORKDIR /app
USER pepic

EXPOSE 8118

ENTRYPOINT ["/app/pepic", "serve", "--config", "/etc/pepic/config.yml"]
