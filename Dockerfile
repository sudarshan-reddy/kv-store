FROM golang:latest as builder
WORKDIR /app

COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/main.go


FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 11200
EXPOSE 11201

# Run the binary program produced by `go build`
CMD ["./main"]
