FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum main.go ./
RUN go mod tidy \
  && CGO_ENABLED=0 go build

FROM alpine:3.20 as runtime

LABEL maintainer="natrontech"

RUN addgroup -S app \
    && adduser -S -G app app

WORKDIR /home/app
COPY --from=builder /build/pbs-exporter .
RUN chown -R app:app ./

USER app

CMD ["./pbs-exporter"]
