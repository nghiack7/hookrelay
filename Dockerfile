FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /bin/hookrelay .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite
COPY --from=builder /bin/hookrelay /bin/hookrelay
RUN mkdir -p /data
ENV DB_PATH=/data/hookrelay.db
ENV PORT=8080
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/bin/hookrelay"]
