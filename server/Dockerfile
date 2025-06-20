FROM golang:1.24.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /server .
COPY schemas ./schemas
COPY message.txt .
EXPOSE 8080
CMD ["/app/server"]