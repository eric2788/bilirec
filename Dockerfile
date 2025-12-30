FROM golang:1.24-alpine AS build

WORKDIR /app
COPY . .

RUN apk add --no-cache tzdata
RUN go run github.com/swaggo/swag/cmd/swag@latest init
RUN go mod download
RUN go mod tidy
RUN go build -v -o bilirec

FROM alpine:latest
WORKDIR /app

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /app/bilirec .
COPY --from=build /app/docs ./docs
RUN chmod +x ./bilirec

ENV TZ=Asia/Hong_Kong

# VOLUMES [ "/app/secrets", "/app/records" ]
# EXPOSE 8080

ENTRYPOINT ["./bilirec"]
