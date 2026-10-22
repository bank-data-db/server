FROM golang:1.26 AS builder

WORKDIR /usr/src/app

RUN mkdir /usr/local/bin/build-output

COPY go.mod go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/build-output ./...

FROM scratch

WORKDIR /cmd
COPY --from=builder /usr/local/bin/build-output .
