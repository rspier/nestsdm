FROM golang:1.16-alpine AS builder

WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN mkdir /tmp/bin && go build -o /tmp/bin ./cmd/*

FROM alpine:3.14

RUN apk add ffmpeg

COPY --from=builder /tmp/bin/* /usr/local/bin/
COPY --from=builder /src/entrypoint.sh /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]