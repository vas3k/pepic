FROM golang:latest AS builder

COPY . /build
WORKDIR /build

RUN go get
RUN go build -a -o /build/pepic .

FROM alpine:latest

RUN apk --no-cache add ca-certificates ffmpeg

COPY --from=builder /build/pepic /pepic
WORKDIR /

EXPOSE 8118

ENTRYPOINT ["/pepic"]
