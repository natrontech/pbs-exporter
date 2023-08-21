FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum main.go ./
RUN go mod tidy \
  && CGO_ENABLED=0 go build

FROM alpine as runtime
COPY --from=builder /build/pbs-exporter /app/pbs-exporter
EXPOSE 9101
CMD ["/app/pbs-exporter"]
