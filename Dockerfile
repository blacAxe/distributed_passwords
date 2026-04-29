FROM golang:1.22

WORKDIR /app
COPY . .
RUN go mod download

# Memberlist uses these ports
EXPOSE 7946/tcp 7946/udp

CMD ["go", "run", "main.go"]