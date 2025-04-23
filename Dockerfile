FROM golang:1.24.2-alpine

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod tidy

COPY . .

CMD ["go", "run", "main.go"]
