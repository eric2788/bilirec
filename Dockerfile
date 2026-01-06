FROM golang:1.25-alpine AS build

WORKDIR /app

RUN apk add --no-cache tzdata

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/swaggo/swag/cmd/swag@v1.16.6

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    /go/bin/swag init -g internal/modules/rest/rest.go -o docs

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o bilirec

FROM alpine:latest
WORKDIR /app

RUN apk update && \
    apk add --no-cache ffmpeg \
    && rm -rf /var/cache/apk/*

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /app/bilirec .
COPY --from=build /app/docs ./docs
RUN chmod +x ./bilirec

ENV TZ=Asia/Hong_Kong

ENV ANONYMOUS_LOGIN=false \
    PORT=8080 \
    MAX_CONCURRENT_RECORDINGS=3 \
    MAX_RECORDING_HOURS=5 \
    MAX_RECOVERY_ATTEMPTS=5 \
    OUTPUT_DIR=records \
    DATABASE_DIR=database \
    CONVERT_FLV_TO_MP4=false \
    DELETE_FLV_AFTER_CONVERT=false

ENV GOMEMLIMIT=256MiB
ENV GOGC=50

ENTRYPOINT ["./bilirec"]