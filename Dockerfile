FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd/web

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /srv
COPY --from=build /app /srv/app
COPY web /srv/web

EXPOSE 8080
CMD ["/srv/app"]
