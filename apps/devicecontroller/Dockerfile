FROM golang:1.22-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o devicecontroller

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/devicecontroller .
RUN apk --no-cache add ca-certificates
EXPOSE 8082 41200
CMD ["./devicecontroller"]
