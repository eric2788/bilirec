FROM golang:1.24-alpine AS build

WORKDIR /app
COPY . .

RUN apk add --no-cache tzdata
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12 \
 && export PATH=$PATH:$(go env GOPATH)/bin \
 && swag init -g internal/modules/rest/rest.go -o docs
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
