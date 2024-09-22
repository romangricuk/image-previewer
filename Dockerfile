# Собираем в гошке
FROM golang:1.22 as build

ENV CODE_DIR /go/src/

WORKDIR ${CODE_DIR}

# Кэшируем слои с модулями
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . ${CODE_DIR}

# Собираем статические бинарники Go (без зависимостей на Си API),
# иначе они не будут работать в alpine образе.
ARG LDFLAGS
RUN CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o /opt/image-previewer-app cmd/image-previewer/main.go

# На выходе тонкий образ
FROM alpine:3.9

# Установка curl для healthcheck
RUN apk --no-cache add curl

# Копируем все собранные бинарники
COPY --from=build /opt/image-previewer-app "/opt/image-previewer-app"