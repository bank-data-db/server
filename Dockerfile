FROM golang:1.26 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/app ./...

FROM scratch

WORKDIR /
COPY --from=builder /usr/local/bin/app /app
