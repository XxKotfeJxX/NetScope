# syntax=docker/dockerfile:1.7
FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/netscope ./cmd/netscope

FROM alpine:3.23
RUN apk add --no-cache ca-certificates \
    && addgroup -S netscope \
    && adduser -S -G netscope netscope
COPY --from=build /out/netscope /usr/local/bin/netscope
USER netscope
EXPOSE 8080
ENTRYPOINT ["netscope"]

