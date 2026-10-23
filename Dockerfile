FROM golang:1.26.10-alpine3.24 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -tags netgo -ldflags '-s -w' -o /out/padelo ./cmd/padelo

FROM alpine:3.24

RUN apk add --no-cache --upgrade ca-certificates \
    && addgroup -S padelo \
    && adduser -S -G padelo -h /app padelo

WORKDIR /app
COPY --from=build /out/padelo ./padelo
COPY --from=build /src/web ./web

USER padelo
EXPOSE 8080

CMD ["./padelo"]
