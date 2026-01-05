FROM golang:1.25-alpine AS build

WORKDIR /app

RUN apk add --no-cache tzdata

ENV GOMODCACHE=/gomod-cache
ENV GOCACHE=/go-cache

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/gomod-cache \
    go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6

RUN --mount=type=cache,target=/gomod-cache \
    --mount=type=cache,target=/go-cache \
    $(go env GOPATH)/bin/swag init -g internal/modules/rest/rest.go -o docs

RUN --mount=type=cache,target=/gomod-cache \
    --mount=type=cache,target=/go-cache \
    go build -v -o bilirec

FROM alpine:latest
WORKDIR /app

RUN apk update && apk add --no-cache ffmpeg

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /app/bilirec .
COPY --from=build /app/docs ./docs
RUN chmod +x ./bilirec

ENV TZ=Asia/Hong_Kong

# Application environment variables (can be overridden at runtime)
# Defaults reflect values used in internal/modules/config/provider
# Secret Values are hidded please refer to internal/modules/config/provider
ENV ANONYMOUS_LOGIN=false \
    PORT=8080 \
    MAX_CONCURRENT_RECORDINGS=3 \
    MAX_RECORDING_HOURS=5 \
    MAX_RECOVERY_ATTEMPTS=5 \
    OUTPUT_DIR=records \
    CONVERT_FLV_TO_MP4=false \
    DELETE_FLV_AFTER_CONVERT=false

ENV GOMEMLIMIT=256MiB
ENV GOGC=50

# VOLUMES [ "/app/secrets", "/app/records" ]
# EXPOSE 8080

ENTRYPOINT ["./bilirec"]
