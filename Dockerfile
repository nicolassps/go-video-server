FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -o video-server .

FROM alpine:latest

RUN apk add --no-cache ffmpeg

WORKDIR /usr/local/bin

COPY --from=builder /app/video-server ./video-server
COPY ./config.yaml ./config.yaml

EXPOSE 8080
CMD ["video-server"]
