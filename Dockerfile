FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tv-guide ./cmd/tv-guide

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /tv-guide /app/tv-guide
ENV PORT=8080
ENV CACHE_PATH=/data/cache/xmltv.xml
EXPOSE 8080
VOLUME ["/data/cache"]
ENTRYPOINT ["/app/tv-guide"]
