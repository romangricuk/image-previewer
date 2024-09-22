FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o image-previewer ./cmd/image-previewer

FROM scratch

WORKDIR /app

COPY --from=builder /app/image-previewer .

CMD ["./image-previewer"]
