FROM golang:latest AS builder

COPY . /build
WORKDIR /build

RUN go get
RUN CGO_ENABLED=0 go build -a -o /build/pepic .

FROM alpine:latest

RUN apk --no-cache add ca-certificates ffmpeg

COPY --from=builder /build/pepic /app/pepic
COPY config /app/config
COPY templates /app/templates
COPY static /app/static
WORKDIR /app

EXPOSE 8118

ENTRYPOINT ["/app/pepic"]
