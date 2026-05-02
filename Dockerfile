FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
# CGO_ENABLED=0 makes the binary statically linked (more portable)
RUN CGO_ENABLED=0 go build -o cracker .

FROM alpine:latest
WORKDIR /root/
# Copy the binary from the builder stage
COPY --from=builder /app/cracker .
# Recreate the static folder and copy the file
RUN mkdir static
COPY --from=builder /app/static/index.html ./static/

EXPOSE 8080 7946 50051
# Ensure the binary has execution permissions
RUN chmod +x ./cracker
CMD ["./cracker"]