FROM alpine:edge AS builder

ENV GOOS=linux
ENV CGO_CFLAGS_ALLOW="-Xpreprocessor"

RUN apk add --no-cache go gcc g++ vips-dev
COPY . /build
WORKDIR /build
RUN go get
RUN go build -a -o /build/pepic -ldflags="-s -w -h" .

FROM alpine:latest

RUN apk --no-cache add ca-certificates mailcap ffmpeg vips
COPY --from=builder /build/pepic /app/pepic
COPY config /app/config
COPY templates /app/templates
COPY static /app/static
WORKDIR /app

EXPOSE 8118

ENTRYPOINT ["/app/pepic"]
